from "alpine:edge"

copy "glue", "/glue"
cmd "/glue"
flatten
tag "xena/glue"
