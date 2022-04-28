#!/usr/bin/env bash

sed 's/window.YTD.tweet.part0 = //' < tweets.js \
  | jq '.[] | [ select(.tweet.retweeted == false) ] | .[].tweet.full_text' \
  | sed -r 's/\s*\.?@[A-Za-z0-9_]+\s*//g' \
  | grep -v 'RT:' \
  | jq --slurp . \
  | jq -r .[] \
  | sed -e 's!http[s]\?://\S*!!g' \
  | sed '/^$/d' \
  > tweets.txt
