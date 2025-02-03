# Basic Gameserver Wine Base Docker Image

<a href="https://travis-ci.com/github/FragSoc/steamcmd-wine-xvfb-docker"><img alt="Travis (.com)" src="https://img.shields.io/travis/com/FragSoc/steamcmd-wine-xvfb-docker?style=flat-square"/></a>
<a href="https://hub.docker.com/r/fragsoc/steamcmd-wine-xvfb"><img alt="Docker Pulls" src="https://img.shields.io/docker/pulls/fragsoc/steamcmd-wine-xvfb?style=flat-square"/></a>
<a href="https://github.com/FragSoc/steamcmd-wine-xvfb-docker"><img alt="GitHub" src="https://img.shields.io/github/license/FragSoc/steamcmd-wine-xvfb-docker?style=flat-square"/></a>

This is a minimal base image based on Debian bookworm, uploaded to docker hub, with [wine](https://www.winehq.org/), [xvfb](https://www.x.org/releases/X11R7.6/doc/man/man1/Xvfb.1.xhtml), [tini](https://github.com/krallin/tini) and [steamcmd](https://developer.valvesoftware.com/wiki/SteamCMD) installed.
It's intended to be used to run conventionally windows-only servers under linux inside docker containers.

Note: `tini` is included because `xvfb-run` won't correctly attach stdout among other issues if run as the root process.

Based on the [`steamcmd/steamcmd`](https://hub.docker.com/r/steamcmd/steamcmd) image.

## Usage

1. Base your docker image upon this one
1. Perform any necessary setup
1. Set your run `CMD` to the server startup command.

```dockerfile
FROM fragsoc/steamcmd-wine-xvfb

# Do some setup RUN commands, call steamcmd etc

CMD "./MyServer.exe"
```

If you need to, you can override the default `ENTRYPOINT` ([`/usr/bin/launch_server`](launch_server.sh)) with your own combination of tools. 

## Similar Tools

- https://github.com/nuxy/docker-steamcmd-wine as recommended on the valve developer wiki.
