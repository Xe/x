#!/bin/sh

CGO_ENABLED=0 GOOS=linux go build -o sona-pi-toki-pona
appsluggr -fname sona-pi-toki-pona.tar.gz -worker sona-pi-toki-pona
scp sona-pi-toki-pona.tar.gz xena@greedo.xeserv.us:public_html/files
ssh dokku@minipaas.xeserv.us tar:from sona-pi-toki-pona https://xena.greedo.xeserv.us/files/sona-pi-toki-pona.tar.gz

