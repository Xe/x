furrybb
======

This boosts any toots with the tag `#furry`, but this can be used for other
hashtags too. Usage is simple:

```console
$ go get github.com/Xe/x/mastodon/furrybb
$ cd $GOPATH/src/github.com/Xe/x/mastodon/furrybb
$ go get github.com/Xe/x/mastodon/mkapp
$ mkapp
Usage of [mkapp]:
  -app-name string
    	app name for mastodon (default "Xe/x bot")
  -instance string
    	mastodon instance
  -password string
    	password to generate token
  -redirect-uri string
    	redirect URI for app users (default "urn:ietf:wg:oauth:2.0:oob")
  -username string
    	username to generate token
  -website string
    	website for users that click the app name (default "https://github.com/Xe/x")
exit status 2
$ mkapp [your options here] > .env
$ echo "HASHTAG=furry" >> .env
$ go build && ./furrybb
```

once you see:

```
time="2017-04-22T19:25:28-07:00" action=streaming.toots
```

you're all good fam.
