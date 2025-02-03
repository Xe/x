#!/usr/bin/env bash

docker build --platform=linux/amd64 -t ghcr.io/xe/steamcmd-wine-xvfb .
docker push ghcr.io/xe/steamcmd-wine-xvfb
