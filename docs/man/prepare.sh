#!/bin/sh

for file in *.1 *.5
do
	mandoc -T markdown $file > ../$file.md
done
