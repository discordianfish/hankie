package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/samalba/dockerclient"
)

const (
	defaultAddr = "unix:///var/run/docker.sock"
)

var (
	addr      = flag.String("a", defaultAddr, "address of docker daemon")
	backupDir = os.Getenv("HOME") + "/.hankie"
	sanitize  = regexp.MustCompile("[^0-9a-zA-Z_-]")
)

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("Subcommand missing")
	}
	cmd := flag.Arg(0)
	args := flag.Args()[1:]
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	switch cmd {
	case "remove":
		/* if fs.NArg() != 1 {
			log.Fatal("Name missing")
		}
		name := fs.Arg(0)
		docker, err := dockerclient.NewDockerClient(*addr)
		if err != nil {
			log.Fatal(err)
		}*/

	case "replace":
		img := fs.String("i", "", "image to run instead")
		conf := fs.String("f", "", "use file instead of getting container from daemon")
		backup := fs.Bool("b", true, "backup container json before removing it")
		keepVolumes := fs.Bool("v", true, "keep volumes")

		if err := fs.Parse(args); err != nil {
			log.Fatal(err)
		}

		if fs.NArg() != 1 {
			log.Fatal("Name missing")

		}

		name := fs.Arg(0)
		docker, err := dockerclient.NewDockerClient(*addr)
		if err != nil {
			log.Fatal(err)
		}

		container := &dockerclient.ContainerInfo{}
		if *conf != "" {
			b, err := ioutil.ReadFile(*conf)
			if err != nil {
				log.Fatal(err)
			}
			if err := json.Unmarshal(b, container); err != nil {
				log.Fatalf("Error parsing %s: %s", *conf, err)
			}
		} else {
			var err error
			container, err = docker.InspectContainer(name)
			if err != nil {
				log.Fatalf("Couldn't get container %s: %s", name, err)
			}

			if *backup {
				os.MkdirAll(backupDir, 0700)
				json, err := json.Marshal(container)
				if err != nil {
					log.Fatal(err)
				}

				daemonName := ""
				if daemonName == defaultAddr {
					daemonName = "local"
				} else {
					daemonName = sanitize.ReplaceAllString(*addr, "-")
				}

				backupFile := fmt.Sprintf("%s/%s_%s_%s.json", backupDir, name, daemonName, time.Now().Format(time.RFC3339))
				if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
					log.Fatalf("Backup file %s already exists", backupFile)
				}

				if err := ioutil.WriteFile(backupFile, json, 0600); err != nil {
					log.Fatal(err)
				}
			}
		}
		if container.Name == "" {
			log.Fatal("replace can only replace named containers")
		}

		if *img != "" {
			container.Config.Image = *img
		}

		image, tag := parseImageName(container.Config.Image)
		if err := docker.PullImage(image, tag); err != nil {
			log.Fatal(err)
		}
		log.Print("- image pulled")

		if err := docker.StopContainer(name, 0); err != nil && err != dockerclient.ErrNotFound {
			log.Fatal(err)
		}
		log.Print("- container stopped")

		if *keepVolumes {

		}

		if err := docker.RemoveContainer(name); err != nil && err != dockerclient.ErrNotFound {
			log.Fatal(err)
		}
		log.Print("- container removed")

		id, err := docker.CreateContainer(container.Config, container.Name[1:]) // remove leading /
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("- container %s created", id)

		log.Printf("Port mapping: %#v", container.HostConfig.PortBindings)
		if err := docker.StartContainer(id, container.HostConfig); err != nil {
			log.Fatal(err)
		}
		log.Print("- container started")

	default:
		log.Printf("Command %s not found", cmd)
		flag.Usage()
		return
	}
}

// Parses image name including an tag and returns image name and tag
// TODO: Future Docker versions can parse the tag on daemon side, see:
// https://github.com/dotcloud/docker/issues/6876
// So this can be deprecated at some point.
func parseImageName(image string) (string, string) {
	tag := ""
	parts := strings.SplitN(image, "/", 2)
	repo := ""
	if len(parts) == 2 {
		repo = parts[0]
		image = parts[1]
	}
	parts = strings.SplitN(image, ":", 2)
	if len(parts) == 2 {
		image = parts[0]
		tag = parts[1]
	}
	if repo != "" {
		image = fmt.Sprintf("%s/%s", repo, image)
	}
	return image, tag
}
