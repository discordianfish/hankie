# Hankie
[Hankie ist ein Dockarbeiter](http://www.youtube.com/watch?v=K5f87QQjcbc)

Hankie is a dock worker.

## Commands
Syntax:

    hankie [global flags] command [command flags]

### Replace
This command will replace a existing container by:

1. reading the current container state from container or file (see -f flag)
2. Pull latest image
3. Stop & remove container
4. Create & start new container

If you need to change the image, use flag -i

#### Flags

      -b=true: backup container json before removing it
      -f="": use file instead of getting container from daemon
      -i="": image to run instead

