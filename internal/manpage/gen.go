// Package manpage is a manpage generator based on command line flags from package flag.
package manpage

import (
	"flag"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DateFormat is the date format used in manpages.
const DateFormat = "January 02, 2006"

// Spew spews out a manpage template for this program then stops execution.
func Spew() {
	var result struct {
		Flags []*flag.Flag
		Name  string
		UName string
		Date  string
	}

	result.Name = filepath.Base(os.Args[0])
	result.UName = strings.ToUpper(result.Name)
	result.Date = time.Now().Format(DateFormat)

	flag.VisitAll(func(f *flag.Flag) {
		result.Flags = append(result.Flags, f)
	})

	var err error
	t := template.New("manpage.1")
	t, err = t.Parse(manpageTemplate)
	if err != nil {
		log.Fatal(err)
	}

	err = t.Execute(os.Stdout, result)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}

const manpageTemplate = `.Dd {{.Date}}
.Dt {{ .UName }} 1 URM


.Sh NAME
.Nm {{ .Name }}
.Nd This is a command that needs a description.


.Sh SYNOPSIS
.Nm
{{ range .Flags }}
.Op Fl {{ .Name }}
{{ end }}


.Sh DESCRIPTION
.Nm
is

TODO: FIXME

.Bl -tag -width " " -offset indent -compact

{{ range .Flags }}
.It Fl {{ .Name }}
{{ .Usage }}

The default value for this is {{ .DefValue }}
{{ end }}

.El


.Sh EXAMPLES

.Li {{ .Name }}

.Li {{ .Name }} -license


.Sh DIAGNOSTICS

.Ex -std {{ .Name }}


.Sh SEE ALSO

.Bl -bullet

.It
.Lk hyperlink: http://some.domain Some Text

.El
`
