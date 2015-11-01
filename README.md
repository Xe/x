# tools
Various tools of mine in Go

Installing these tools
----------------------

To install any of these tools, type in:

```console
$ go get christine.website/go/tools/$toolname
```

For example:

```console
$ go get christine.website/go/tools/license
```

`dokku`
-------

This is a simple command line tool to interface with Dokku servers. This is
a port of my shell extension
[`dokku.zsh`](https://github.com/Xe/dotfiles/blob/master/.zsh/dokku.zsh) to
a nice Go binary.

This takes a configuration file for defining multiple servers:

```ini
[server "default"]
user = dokku
host = panel.apps.xeserv.us
sshkey = /.ssh/id_rsa
```

By default it will imply that the SSH key is `~/.ssh/id_rsa` and that the
username is `dokku`. By default the server named `default` will be used for
command execution.

### TODO

- [ ] Allow interactive commands
- [ ] Directly pipe stdin and stdout to the ssh connection

---

`license`
---------

This is a simple command line tool to help users generate a license file based 
on information they have already given their system and is easy for the system 
to figure out on its own.

```console
$ license
Usage of license:
license [options] <license kind>

  -email="": email of the person licensing the software
  -name="": name of the person licensing the software
  -out=false: write to a file instead of stdout
  -show=false: show all licenses instead of generating one

By default the name and email are scraped from `git config`
```

```console
$ license -show
Licenses available:
  zlib
  unlicense
  mit
  apache
  bsd-2
  gpl-2
```

```console
$ license zlib
Copyright (c) 2015 Christine Dodrill <xena@yolo-swag.com>

This software is provided 'as-is', without any express or implied
warranty. In no event will the authors be held liable for any damages
arising from the use of this software.

Permission is granted to anyone to use this software for any purpose,
including commercial applications, and to alter it and redistribute it
freely, subject to the following restrictions:

1. The origin of this software must not be misrepresented; you must not
   claim that you wrote the original software. If you use this software
   in a product, an acknowledgement in the product documentation would be
   appreciated but is not required.

2. Altered source versions must be plainly marked as such, and must not be
   misrepresented as being the original software.

3. This notice may not be removed or altered from any source distribution.
```

---

`ghstat`
--------

Command ghstat shows the status of GitHub via their status API.

    Usage of ./ghstat:
      -message=false: show last message?

This follows https://status.github.com/api for all but the list of all recent
status messages.

```console
$ ghstat
Status: minor (Fri Mar 27 15:24:57 2015)
```

```console
$ ghstat -message
Last message:
Status:  minor
Message: We've deployed our volumetric attack defenses against an extremely 
large amount of traffic. Performance is stabilizing.
Time:    Fri Mar 27 15:04:59 2015
```
