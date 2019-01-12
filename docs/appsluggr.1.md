APPSLUGGR(1) - General Commands Manual (urm)

# NAME

**appsluggr** - appsluggr packages a precompiled binary application as a Heroku style slug for use with Dokku.

# SYNOPSIS

**appsluggr**

\[**-fname**]
\[**-license**]
\[**-web**]
\[**-web-scale**]
\[**-worker**]
\[**-worker-scale**]

# DESCRIPTION

**appsluggr**
is a small tool to package
`GOOS=linux GOARCH=amd64`
binaries for consumption on
`hyperlink: http://dokku.viewdocs.io/dokku/ Dokku`

**-fname**

> The filename to write the resulting slug to.

> The default value for this is
> `slug.tar.gz`

**-license**

> If set, the tool will show its software license details and then exit.

**-web**

> The path to the binary for the web process.

> One of
> **-web**
> or
> **-worker**
> must be set.

**-web-scale**

> The default scale for web process if defined.

> The default value for this is 1.

**-worker**

> The path to the binary for the worker process.
> One of
> **-web**
> or
> **-worker**
> must be set.

**-worker-scale**

> The default scale for the worker process if defined.

> The default value for this is 1

# EXAMPLES

`appsluggr`

`appsluggr -web web`

`appsluggr -worker ilo-sona`

`appsluggr -fname foo.tar.gz -web web -worker worker -web-scale 4 -worker-scale 16`

# IMPLEMENTATION NOTES

**appsluggr**
when used with
[http://dokku.viewdocs.io/dokku/ Dokku](hyperlink:)
requires the use of the
[https://github.com/ryandotsmith/null-buildpack Null Buildpack](hyperlink:)
as follows:

`$ dokku config:set $APP_NAME BUILDPACK_URL=https://github.com/ryandotsmith/null-buildpack`

Or

`$ ssh dokku@host config:set <see above>`

# DIAGNOSTICS

The **appsluggr** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

*	[http://dokku.viewdocs.io/dokku/ Dokku](hyperlink:)

*	[https://github.com/ryandotsmith/null-buildpack Null Buildpack](hyperlink:)

 \- December 9, 2018
