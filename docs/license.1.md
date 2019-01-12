LICENSE(1) - General Commands Manual (urm)

# NAME

**license** - Generate software licenses from a rather large list of templates.

# SYNOPSIS

**license**
\[**-email**&nbsp;*address*]
\[**-license**]
\[**-name**&nbsp;*name*]
\[**-out**]
\[**-show**]

# DESCRIPTION

**license**
is a software license generator. It uses
git-config(1)
to parse out your email and "real name" when relevant for the license template reasons.

**-email** *address*

> The email of the person licensing the software. This should be your email, or a corporation's email. If in doubt, ask a lawyer what to put here.

> The default value for this is derived from
> git-config(1)
> by using the command:

> `$ git config user.email`

**-license**

> If set,
> **license**
> will show its software license details and then exit.

**-name** *name*

> The name of the person licensing the software. This should be your name, or a corporation's name. If in doubt, ask a lawyer what to put here.

> The default value for this is derived from
> git-config(1)
> by using the command:

> `$ git config user.name`

**-out**

> If this is set,
> **license**
> will write the resulting license to the disk instead of standard out.

**-show**

> If set,
> **license**
> will show its list of license templates instead of generating one.

# EXAMPLES

`license`

`license -license`

`license -show`

`license mit`

# DIAGNOSTICS

The **license** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

 \- December 9, 2018
