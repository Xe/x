package flagenv_test

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/Xe/x/internal/flagenv"
	"github.com/facebookgo/ensure"
)

func named(t, v string) string { return strings.ToUpper(t + v) }

func ExampleParse() {
	var raz string
	flag.StringVar(&raz, "raz-value", "bar", "set the raz")

	// override default flag value with value found in MY_RAZ_VALUE
	flagenv.Prefix = "my_"
	flagenv.Parse()

	// override value found in MY_RAZ_VALUE with command line flag value -raz-value=foo
	flag.Parse()
}

func TestNothingToDo(t *testing.T) {
	const name = "TestNothingToDo"
	s := flag.NewFlagSet(name, flag.PanicOnError)
	s.String(named(name, "foo"), "", "")
	ensure.Nil(t, flagenv.ParseSet(name, s))
}

func TestAFewFlags(t *testing.T) {
	const name = "TestAFewFlags"
	s := flag.NewFlagSet(name, flag.PanicOnError)
	const foo = "42"
	const bar = int(43)
	fooActual := s.String("foo", "", "")
	barActual := s.Int("bar", 0, "")
	os.Setenv(named(name, "foo"), foo)
	os.Setenv(named(name, "bar"), fmt.Sprint(bar))
	ensure.Nil(t, flagenv.ParseSet(name, s))
	ensure.DeepEqual(t, *fooActual, foo)
	ensure.DeepEqual(t, *barActual, bar)
}

func TestInvalidFlagValue(t *testing.T) {
	const name = "TestInvalidFlagValue"
	s := flag.NewFlagSet(name, flag.PanicOnError)
	s.Int("bar", 0, "")
	os.Setenv(named(name, "bar"), "a")
	ensure.Err(t, flagenv.ParseSet(name, s),
		regexp.MustCompile(`failed to set flag "bar" with value "a"`))
}

func TestReturnsFirstError(t *testing.T) {
	const name = "TestReturnsFirstError"
	s := flag.NewFlagSet(name, flag.PanicOnError)
	s.Int("bar1", 0, "")
	s.Int("bar2", 0, "")
	os.Setenv(named(name, "bar1"), "a")
	ensure.Err(t, flagenv.ParseSet(name, s),
		regexp.MustCompile(`failed to set flag "bar1" with value "a"`))
}

func TestExplicitAreIgnored(t *testing.T) {
	const name = "TestExplicitAreIgnored"
	s := flag.NewFlagSet(name, flag.PanicOnError)
	const bar = int(43)
	barActual := s.Int("bar", 0, "")
	s.Parse([]string{"-bar", fmt.Sprint(bar)})
	os.Setenv(named(name, "bar"), "44")
	ensure.Nil(t, flagenv.ParseSet(name, s))
	ensure.DeepEqual(t, *barActual, bar)
}

func TestGlobalParse(t *testing.T) {
	flagenv.Parse()
}
