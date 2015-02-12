Dokku
=====

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

TODO
----

- [ ] Allow interactive commands
- [ ] Directly pipe stdin and stdout to the ssh connection
