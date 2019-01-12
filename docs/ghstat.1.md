GHSTAT(1) - General Commands Manual (urm)

# NAME

**ghstat** - Look up and summarize the status of GitHub.

# SYNOPSIS

**ghstat**
\[**-license**]
\[**-message**]

# DESCRIPTION

**ghstat**
is a small tool to help users look up information about
[https://github.com GitHub](hyperlink:)
system status as viewed by their
[https://status.github.com Status API](hyperlink:)

By default this tool will print a very small summary of GitHub status followed by the time the last update was made in RFC 3339 time format.

Here's an example:

`$ ghstat`
`Status: good (2018-12-06T17:09:57Z)`

**-license**

> If set, the tool will show its software license details and then exit.

**-message**

> If set, the tool will show the last status message from GitHub more verbosely like such:

> `$ ghstat -message`
> `Last message:`
> `Status: good`
> `Message:`
> `Time:`

> When there is a message relevant to the status, it and its time will be shown here.

# EXAMPLES

`ghstat`

`ghstat -license`

`ghstat -message`

# DIAGNOSTICS

The **ghstat** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

*	[https://github.com GitHub](hyperlink:)

*	[https://status.github.com GitHub Status](hyperlink:)

 \- December 9, 2018
