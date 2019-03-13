package i18n

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestLingo(t *testing.T) {
	l := New("de_DE", "translations")
	t1 := l.TranslationsForLocale("en_US")
	r1 := t1.Value("main.subtitle")
	r1Exp := "Knives that put cut in cutlery."
	if r1 != r1Exp {
		t.Errorf("Expected \""+r1Exp+"\", got %s", r1)
		t.Fail()
	}
	r2 := t1.Value("home.title")
	r2Exp := "Welcome to CutleryPlus!"
	if r2 != r2Exp {
		t.Errorf("Expected \""+r2Exp+"\", got %s", r2)
		t.Fail()
	}
	r3 := t1.Value("menu.products.self")
	r3Exp := "Products"
	if r3 != r3Exp {
		t.Errorf("Expected \""+r3Exp+"\", got %s", r3)
		t.Fail()
	}
	r4 := t1.Value("menu.non.existant")
	r4Exp := "non.existant"
	if r4 != r4Exp {
		t.Errorf("Expected \""+r4Exp+"\", got %s", r4)
		t.Fail()
	}
	r5 := t1.Value("error.404", "idnex.html")
	r5Exp := "Page idnex.html not found!"
	if r5 != r5Exp {
		t.Errorf("Expected \""+r5Exp+"\", got \"%s\"", r5)
		t.Fail()
	}
}

func TestLingoHttp(t *testing.T) {
	l := New("en_US", "translations")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := r.Header.Get("Expected-Results")
		t1 := l.TranslationsForRequest(r)
		r1 := t1.Value("error.500")
		if r1 != expected {
			t.Errorf("Expected \""+expected+"\", got %s", r1)
			t.Fail()
		}
	}))
	defer srv.Close()
	url, _ := url.Parse(srv.URL)

	req1 := &http.Request{
		Method: "GET",
		Header: map[string][]string{
			"Accept-Language":  {"sr, en-gb;q=0.8, en;q=0.7"},
			"Expected-Results": {"Greska sa nase strane, pokusajte ponovo."},
		},
		URL: url,
	}
	req2 := &http.Request{
		Method: "GET",
		Header: map[string][]string{
			"Accept-Language":  {"en-US, en-gb;q=0.8, en;q=0.7"},
			"Expected-Results": {"Something is wrong on our side, please try again."},
		},
		URL: url,
	}
	req3 := &http.Request{
		Method: "GET",
		Header: map[string][]string{
			"Accept-Language":  {"de-at, en-gb;q=0.8, en;q=0.7"},
			"Expected-Results": {"Stimmt etwas nicht auf unserer Seite ist, versuchen Sie es erneut."},
		},
		URL: url,
	}

	http.DefaultClient.Do(req1)
	http.DefaultClient.Do(req2)
	http.DefaultClient.Do(req3)
}
