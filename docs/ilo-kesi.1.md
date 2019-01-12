ILO-KESI(1) - General Commands Manual (urm)

# NAME

**ilo-kesi** - ni li ilo sona pi toki pona.

# SYNOPSIS

**ilo-kesi**
\[**-repl**&nbsp;*TOKEN*]

# DESCRIPTION

**ilo-kesi**
communicates with Discord and scans every message in every channel it is in for the following pattern:

`ilo ${ILO_NIMI} o`

This is usually:

`ilo Kesi o`

When this condition is met, the chat message will be posted to the
`TOKI_PONA_TOKENIZER_API_URL`
and the resulting parsed sentences will be analyzed for what is being asked, and then it will be done.

This only works on sentences written in the
[http://tokipona.org Toki Pona](hyperlink)
constructed language.

**-repl** *REPL*

> When this flag is passed,
> **ilo-kesi**
> will function in a mode where it does not connect to discord. This is useful when debugging parts of the grammar parsing. You can pass a junk value to
> `DISCORD_TOKEN`
> to help make testing easier.

# ENVIRONMENT

`DISCORD_TOKEN`

> Specifies the Discord token that
> **ilo-kesi**
> will use for client communication.

`TOKI_PONA_TOKENIZER_API_URL`

> Specifies the URL that
> **ilo-kesi**
> will use to tokenize Toki Pona sentences. This should be some instance of the following serverless function:

> [https://github.com/Xe/x/blob/master/discord/ilo-kesi/function/index.js](hyperlink:)

> The default value for this is:

> [https://us-central1-golden-cove-408.cloudfunctions.net/function-1](hyperlink:)

`SWITCH_COUNTER_WEBHOOK`

> Specifies the URL that
> **ilo-kesi**
> will use to communicate with
> [https://www.switchcounter.science Switch Counter](hyperlink:)
> This will be used mainly to read data, unless the user in question is a member of the
> `JAN_LAWA`
> id set.

`ILO_NIMI`

> Specifies the name of
> **ilo-kesi**
> when being commanded to do stuff. This defaults to
> `Kesi`

JAN\_LAWA

> Specifies the list of people (via Discord user ID's) that are allowed to use
> **ilo-kesi**
> to submit switch data to
> [https://www.switchcounter.science Switch Counter](hyperlink:)

# IMPLEMENTATION NOTES

**ilo-kesi**
requires a brain created by
cadeybot(1)

**ilo-kesi**
requires a webhook from
[https://www.switchcounter.science Switch Counter](hyperlink:)

# EXAMPLES

ilo-kesi

ilo-kesi -repl

# DIAGNOSTICS

The **ilo-kesi** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

*	[https://discordapp.com Discord](hyperlink:)

*	[http://tokipona.org Toki Pona](hyperlink)

*	[https://www.switchcounter.science Switch Counter](hyperlink:)

 \- December 19, 2018
