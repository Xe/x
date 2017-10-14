package gluahttp

import "github.com/yuin/gopher-lua"
import "testing"
import "io/ioutil"
import "net/http"
import "net"
import "fmt"
import "net/http/cookiejar"
import "strings"

func TestRequestNoMethod(t *testing.T) {
	if err := evalLua(t, `
		local http = require("http")
		response, error = http.request()

		assert_equal(nil, response)
		assert_contains('unsupported protocol scheme ""', error)
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestRequestNoUrl(t *testing.T) {
	if err := evalLua(t, `
		local http = require("http")
		response, error = http.request("get")

		assert_equal(nil, response)
		assert_contains('unsupported protocol scheme ""', error)
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestRequestBatch(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		responses, errors = http.request_batch({
			{"get", "http://`+listener.Addr().String()+`", {query="page=1"}},
			{"post", "http://`+listener.Addr().String()+`/set_cookie"},
			{"post", ""},
			1
		})

		assert_equal(nil, errors[1])
		assert_equal(nil, errors[2])
		assert_contains('unsupported protocol scheme ""', errors[3])
		assert_equal('Request must be a table', errors[4])

		assert_equal('Requested GET / with query "page=1"', responses[1]["body"])
		assert_equal('Cookie set!', responses[2]["body"])
		assert_equal('12345', responses[2]["cookies"]["session_id"])
		assert_equal(nil, responses[3])
		assert_equal(nil, responses[4])

		responses, errors = http.request_batch({
			{"get", "http://`+listener.Addr().String()+`/get_cookie"}
		})

		assert_equal(nil, errors)
		assert_equal("session_id=12345", responses[1]["body"])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestRequestGet(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.request("get", "http://`+listener.Addr().String()+`")

		assert_equal('Requested GET / with query ""', response['body'])
		assert_equal(200, response['status_code'])
		assert_equal('29', response['headers']['Content-Length'])
		assert_equal('text/plain; charset=utf-8', response['headers']['Content-Type'])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestRequestGetWithRedirect(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.request("get", "http://`+listener.Addr().String()+`/redirect")

		assert_equal('Requested GET / with query ""', response['body'])
		assert_equal(200, response['status_code'])
		assert_equal('http://`+listener.Addr().String()+`/', response['url'])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestRequestPostForm(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.request("post", "http://`+listener.Addr().String()+`", {
			form="username=bob&password=secret"
		})

		assert_equal(
			'Requested POST / with query ""' ..
			'Content-Type: application/x-www-form-urlencoded' ..
			'Content-Length: 28' ..
			'Body: username=bob&password=secret', response['body'])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestRequestHeaders(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.request("post", "http://`+listener.Addr().String()+`", {
			headers={
				["Content-Type"]="application/json"
			}
		})

		assert_equal(
			'Requested POST / with query ""' ..
			'Content-Type: application/json' ..
			'Content-Length: 0' ..
			'Body: ', response['body'])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestRequestQuery(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.request("get", "http://`+listener.Addr().String()+`", {
			query="page=2"
		})

		assert_equal('Requested GET / with query "page=2"', response['body'])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestGet(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.get("http://`+listener.Addr().String()+`", {
			query="page=1"
		})

		assert_equal('Requested GET / with query "page=1"', response['body'])
		assert_equal(200, response['status_code'])
		assert_equal('35', response['headers']['Content-Length'])
		assert_equal('text/plain; charset=utf-8', response['headers']['Content-Type'])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestDelete(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.delete("http://`+listener.Addr().String()+`", {
			query="page=1"
		})

		assert_equal('Requested DELETE / with query "page=1"', response['body'])
		assert_equal(200, response['status_code'])
		assert_equal('38', response['headers']['Content-Length'])
		assert_equal('text/plain; charset=utf-8', response['headers']['Content-Type'])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestHead(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.head("http://`+listener.Addr().String()+`/head", {
			query="page=1"
		})

		assert_equal(200, response['status_code'])
		assert_equal("/head?page=1", response['headers']['X-Request-Uri'])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestPost(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.post("http://`+listener.Addr().String()+`", {
			body="username=bob&password=secret"
		})

		assert_equal(
			'Requested POST / with query ""' ..
			'Content-Type: ' ..
			'Content-Length: 28' ..
			'Body: username=bob&password=secret', response['body'])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestPatch(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.patch("http://`+listener.Addr().String()+`", {
			body='{"username":"bob"}',
			headers={
				["Content-Type"]="application/json"
			}
		})

		assert_equal(
			'Requested PATCH / with query ""' ..
			'Content-Type: application/json' ..
			'Content-Length: 18' ..
			'Body: {"username":"bob"}', response['body'])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestPut(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.put("http://`+listener.Addr().String()+`", {
			body="username=bob&password=secret",
			headers={
				["Content-Type"]="application/x-www-form-urlencoded"
			}
		})

		assert_equal(
			'Requested PUT / with query ""' ..
			'Content-Type: application/x-www-form-urlencoded' ..
			'Content-Length: 28' ..
			'Body: username=bob&password=secret', response['body'])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestResponseCookies(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.get("http://`+listener.Addr().String()+`/set_cookie")

		assert_equal('Cookie set!', response["body"])
		assert_equal('12345', response["cookies"]["session_id"])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestRequestCookies(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.get("http://`+listener.Addr().String()+`/get_cookie", {
			cookies={
				["session_id"]="test"
			}
		})

		assert_equal('session_id=test', response["body"])
		assert_equal(15, response["body_size"])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestResponseBodySize(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.get("http://`+listener.Addr().String()+`/")

		assert_equal(29, response["body_size"])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestResponseBody(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")
		response, error = http.get("http://`+listener.Addr().String()+`/")

		assert_equal("Requested XXX / with query \"\"", string.gsub(response.body, "GET", "XXX"))
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func TestResponseUrl(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	setupServer(listener)

	if err := evalLua(t, `
		local http = require("http")

		response, error = http.get("http://`+listener.Addr().String()+`/redirect")
		assert_equal("http://`+listener.Addr().String()+`/", response["url"])

		response, error = http.get("http://`+listener.Addr().String()+`/get_cookie")
		assert_equal("http://`+listener.Addr().String()+`/get_cookie", response["url"])
	`); err != nil {
		t.Errorf("Failed to evaluate script: %s", err)
	}
}

func evalLua(t *testing.T, script string) error {
	L := lua.NewState()
	defer L.Close()

	cookieJar, _ := cookiejar.New(nil)

	L.PreloadModule("http", NewHttpModule(&http.Client{
		Jar: cookieJar,
	},
	).Loader)

	L.SetGlobal("assert_equal", L.NewFunction(func(L *lua.LState) int {
		expected := L.Get(1)
		actual := L.Get(2)

		if expected.Type() != actual.Type() || expected.String() != actual.String() {
			t.Errorf("Expected %s %q, got %s %q", expected.Type(), expected, actual.Type(), actual)
		}

		return 0
	}))

	L.SetGlobal("assert_contains", L.NewFunction(func(L *lua.LState) int {
		contains := L.Get(1)
		actual := L.Get(2)

		if !strings.Contains(actual.String(), contains.String()) {
			t.Errorf("Expected %s %q contains %s %q", actual.Type(), actual, contains.Type(), contains)
		}

		return 0
	}))

	return L.DoString(script)
}

func setupServer(listener net.Listener) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Requested %s / with query %q", req.Method, req.URL.RawQuery)

		if req.Method == "POST" || req.Method == "PATCH" || req.Method == "PUT" {
			body, _ := ioutil.ReadAll(req.Body)
			fmt.Fprintf(w, "Content-Type: %s", req.Header.Get("Content-Type"))
			fmt.Fprintf(w, "Content-Length: %s", req.Header.Get("Content-Length"))
			fmt.Fprintf(w, "Body: %s", body)
		}
	})
	mux.HandleFunc("/head", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "HEAD" {
			w.Header().Set("X-Request-Uri", req.URL.String())
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
	mux.HandleFunc("/set_cookie", func(w http.ResponseWriter, req *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "12345"})
		fmt.Fprint(w, "Cookie set!")
	})
	mux.HandleFunc("/get_cookie", func(w http.ResponseWriter, req *http.Request) {
		session_id, _ := req.Cookie("session_id")
		fmt.Fprint(w, session_id)
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/", http.StatusFound)
	})
	s := &http.Server{
		Handler: mux,
	}
	go s.Serve(listener)
}
