flagconfyg(5) - File Formats Manual (urm)

# NAME

**flagconfyg** - This is the simple configuration format for projects in the x repos.

# DESCRIPTION

**flagconfyg**
is a simple configuration language for projects in this repository. They are
configured by a file that looks like this:

// for appsluggr
web web
web-scale 1

fname (
  slug.tar.gz
)

and this gets resolved to the following flag calls:

\-web=web -web-scale=1 -fname=slug.tar.gz

 \- January 12, 2019
