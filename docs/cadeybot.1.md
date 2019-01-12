CADEYBOT(1) - General Commands Manual (urm)

# NAME

**cadeybot** - Markov bot based on my discord GDPR dump.

# SYNOPSIS

**cadeybot**
\[**-token**&nbsp;*TOKEN*]
\[**-brain**&nbsp;*BRAIN*]

# DESCRIPTION

**cadeybot**
is a simple markov chatbot. Mention it in any channel the bot is in to make it spew out amusing text.

`TOKEN` **-token** *TOKEN*

> Specifies the Discord token that
> **cadeybot**
> will use for client communication.

`BRAIN` **-token** *TOKEN*

> Specifies the Markov chain brain that
> **cadeybot**
> should load data into cadey.gob from.

# IMPLEMENTATION NOTES

In order for
**cadeybot**
to get markov bot data, please put the importer tool and corpusmake.sh into the messages folder of your Discord GDPR dump. Then run corpusmake.sh and pass the resulting brain.txt as -brain to
**cadeybot**.

# EXAMPLES

`cadeybot`

`cadeybot -brain brain.txt`

# DIAGNOSTICS

The **cadeybot** utility exits&#160;0 on success, and&#160;&gt;0 if an error occurs.

# SEE ALSO

*	[https://discordapp.com Discord](hyperlink:)

 \- December 19, 2018
