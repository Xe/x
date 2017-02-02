from "xena/go"

workdir "/"
copy "main.go", "/go/src/github.com/Xe/tools/irc/bncadmin/main.go"
copy "vendor", "/go/src/github.com/Xe/tools/irc/bncadmin/"
run "go install github.com/Xe/tools/irc/bncadmin && cp /go/bin/bncadmin /usr/bin/bncadmin"

run "rm -rf /usr/local/go /go && apk del bash gcc musl-dev openssl go"
flatten

cmd "/usr/bin/bncadmin"

tag "xena/bncadmin"
