package gluaurl

import "github.com/yuin/gopher-lua"
import "testing"
import "os"
import "bytes"
import "io"
import "strings"

func TestParse(t *testing.T) {
	output, err := evalScript(`
local url = require("url")

parsed = url.parse("http://bob:secret@example.com:8080/products?page=2#something")

print(parsed.scheme)
print(parsed.username)
print(parsed.password)
print(parsed.host)
print(parsed.path)
print(parsed.query)
print(parsed.fragment)
`)

	if err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	} else {
		if expected := `http
bob
secret
example.com:8080
/products
page=2
something
`; expected != output {
			t.Errorf("Expected output does not match actual output\nExpected: %s\nActual: %s", expected, output)
		}
	}
}

func TestParseOnlyHost(t *testing.T) {
	output, err := evalScript(`
local url = require("url")

parsed = url.parse("https://example.com")

print(parsed.scheme)
print(parsed.username)
print(parsed.password)
print(parsed.host)
print(parsed.path)
print(parsed.query)
print(parsed.fragment)
`)

	if err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	} else {
		if expected := `https
nil
nil
example.com



`; expected != output {
			t.Errorf("Expected output does not match actual output\nExpected: %s\nActual: %s", expected, output)
		}
	}
}

func TestBuild(t *testing.T) {
	output, err := evalScript(`
local url = require("url")

built = url.build({
	scheme="https",
	username="bob",
	password="secret",
	host="example.com:8080",
	path="/products",
	query="page=2",
	fragment="something"
})

print(built)
`)

	if err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	} else {
		if expected := `https://bob:secret@example.com:8080/products?page=2#something
`; expected != output {
			t.Errorf("Expected output does not match actual output\nExpected: %s\nActual: %s", expected, output)
		}
	}
}

func TestBuildEmpty(t *testing.T) {
	output, err := evalScript(`
local url = require("url")

built = url.build({})

print(built)
`)

	if err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	} else {
		if expected := `
`; expected != output {
			t.Errorf("Expected output does not match actual output\nExpected: %s\nActual: %s", expected, output)
		}
	}
}

func TestBuildQueryString(t *testing.T) {
	output, err := evalScript(`
local url = require("url")

function assert_query_string(options, expected, message)
	actual = url.build_query_string(options)

	if expected ~= actual then
		print("Failed to build '" .. message .. "'")
		print("Expected:")
		print(expected)
		print("Actual:")
		print(actual)
	end
end

assert_query_string(
	{foo="bar", baz=42, quux="All your base are belong to us"},
	"baz=42&foo=bar&quux=All+your+base+are+belong+to+us",
	"simple"
)

assert_query_string(
	{someName={1, 2, 3}, regularThing="blah"},
	"regularThing=blah&someName%5B%5D=1&someName%5B%5D=2&someName%5B%5D=3",
	"with array"
)

assert_query_string(
	{foo={"a", "b", "c"}},
	"foo%5B%5D=a&foo%5B%5D=b&foo%5B%5D=c",
	"with array of strings"
)

assert_query_string(
	{foo={"baz", 42, "All your base are belong to us"}},
	"foo%5B%5D=baz&foo%5B%5D=42&foo%5B%5D=All+your+base+are+belong+to+us",
	"more array"
)

assert_query_string(
	{foo={bar="baz", beep=42, quux="All your base are belong to us"}},
	"foo%5Bbar%5D=baz&foo%5Bbeep%5D=42&foo%5Bquux%5D=All+your+base+are+belong+to+us",
	"even more arrays"
)

assert_query_string(
	{a={1,2}, b={c=3, d={4,5}, e={ x={6}, y=7, z={8,9} }, f=true, g=false, h=""}, i={10,11}, j=true, k=false, l={"",0}, m="cowboy hat?" },
	"a%5B%5D=1&a%5B%5D=2&b%5Bc%5D=3&b%5Bd%5D%5B%5D=4&b%5Bd%5D%5B%5D=5&b%5Be%5D%5Bx%5D%5B%5D=6&b%5Be%5D%5By%5D=7&b%5Be%5D%5Bz%5D%5B%5D=8&b%5Be%5D%5Bz%5D%5B%5D=9&b%5Bf%5D=true&b%5Bg%5D=false&b%5Bh%5D=&i%5B%5D=10&i%5B%5D=11&j=true&k=false&l%5B%5D=&l%5B%5D=0&m=cowboy+hat%3F",
	"huge structure"
)

assert_query_string(
	{ a={0, { 1, 2 }, { 3, { 4, 5 }, { 6 } }, { b= { 7, { 8, 9 }, { { c=10, d=11 } }, { { 12 } }, { { { 13 } } }, { e= { f= { g={ 14, { 15 } } } } }, 16 } }, 17 } },
	"a%5B%5D=0&a%5B1%5D%5B%5D=1&a%5B1%5D%5B%5D=2&a%5B2%5D%5B%5D=3&a%5B2%5D%5B1%5D%5B%5D=4&a%5B2%5D%5B1%5D%5B%5D=5&a%5B2%5D%5B2%5D%5B%5D=6&a%5B3%5D%5Bb%5D%5B%5D=7&a%5B3%5D%5Bb%5D%5B1%5D%5B%5D=8&a%5B3%5D%5Bb%5D%5B1%5D%5B%5D=9&a%5B3%5D%5Bb%5D%5B2%5D%5B0%5D%5Bc%5D=10&a%5B3%5D%5Bb%5D%5B2%5D%5B0%5D%5Bd%5D=11&a%5B3%5D%5Bb%5D%5B3%5D%5B0%5D%5B%5D=12&a%5B3%5D%5Bb%5D%5B4%5D%5B0%5D%5B0%5D%5B%5D=13&a%5B3%5D%5Bb%5D%5B5%5D%5Be%5D%5Bf%5D%5Bg%5D%5B%5D=14&a%5B3%5D%5Bb%5D%5B5%5D%5Be%5D%5Bf%5D%5Bg%5D%5B1%5D%5B%5D=15&a%5B3%5D%5Bb%5D%5B%5D=16&a%5B%5D=17",
	"nested arrays"
)

assert_query_string(
	{ a= {1,2,3}, ["b[]"]= {4,5,6}, ["c[d]"]= {7,8,9}, e= { f= {10}, g= {11,12}, h= 13 } },
	"a%5B%5D=1&a%5B%5D=2&a%5B%5D=3&b%5B%5D=4&b%5B%5D=5&b%5B%5D=6&c%5Bd%5D%5B%5D=7&c%5Bd%5D%5B%5D=8&c%5Bd%5D%5B%5D=9&e%5Bf%5D%5B%5D=10&e%5Bg%5D%5B%5D=11&e%5Bg%5D%5B%5D=12&e%5Bh%5D=13",
	"make sure params are not double-encoded"
)

`)

	if err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	} else {
		if expected := ``; expected != output {
			t.Error(output)
		}
	}
}

func TestResolve(t *testing.T) {
	output, err := evalScript(`
local url = require("url")

print(url.resolve('/one/two/three', 'four'))
print(url.resolve('http://example.com/', '/one'))
print(url.resolve('http://example.com/one', '/two'))
print(url.resolve('https://example.com/one', '//example2.com'))
`)

	if err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	} else {
		if expected := `/one/two/four
http://example.com/one
http://example.com/two
https://example2.com
`; expected != output {
			t.Errorf("Expected output does not match actual output\nExpected: %s\nActual: %s", expected, output)
		}
	}
}

func evalScript(script string) (string, error) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("url", Loader)

	var err error

	out := captureStdout(func() {
		err = L.DoString(script)
	})

	return out, err
}

func captureStdout(inner func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	inner()

	w.Close()
	os.Stdout = oldStdout
	out := strings.Replace(<-outC, "\r", "", -1)

	return out
}
