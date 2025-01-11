#!/usr/bin/env bash
version=$(curl -sX GET "https://api.github.com/repos/actions/runner/releases/latest" | jq --raw-output '.tag_name')
version="${version#*v}"
version="${version#*release-}"

docker build --platform linux/amd64 --build-arg VERSION=${version} -t ghcr.io/xe/x/actions-runner .
docker push ghcr.io/xe/x/actions-runner