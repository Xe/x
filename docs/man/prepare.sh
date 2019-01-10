#!/bin/sh

for file in *.1
do
	mandoc -T markdown $file > ../$file.md
done

gzip *.1
