#!/bin/sh

set -e
set -x

for dir in $(find -type d)
do
    [ "$dir" != . ] && ./importer ./"$dir"/messages.csv >> brain.txt
done
