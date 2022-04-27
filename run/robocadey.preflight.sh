#!/usr/bin/env bash

rm tweet_minimal.json
sed 's/window.YTD.tweet.part0 = //' < tweets.js \
  | jq '.[] | [ select(.tweet.retweeted == false) ] | .[].tweet.full_text' \
  | sed -r 's/\s*\.?@[A-Za-z0-9_]+\s*//g' \
  | grep -v 'RT:' \
  | jq --slurp . > tweet_minimal.json
