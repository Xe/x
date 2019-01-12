QUICKSERV(1) - General Commands Manual (prm)

# NAME

**quickserv** - quickly and dirtily serve a folder as a HTTP server.

# SYNOPSIS

**quickserv**
\[**-dir**]
\[**-port**]

# DESCRIPTION

**quickserv**
serves a local directory of files as an HTTP server.

**-dir**

> Specifies the local path to be served over HTTP.

**-port**

> Specifies the TCP port that quickserv will bind to.

# EXAMPLES

`quickserv`

`quickserv -dir ~/public_html -port 9001`

# SEE ALSO

*	[https://godoc.org/net/http#Dir](hyperlink:)

 \- December 12, 2017
