package i18n

import (
	"testing"
)

func TestLocale(t *testing.T) {
	l0 := supportedLocales("ja-JP;q")
	if len(l0) != 1 {
		t.Errorf("Expected number of locales \"1\", got %d", len(l0))
		t.Fail()
	}
	l1 := supportedLocales("en,de-AT; q=0.8,de;q=0.6,bg; q=0.4,en-US;q=0.2,sr;q=0.2")
	if len(l1) != 6 {
		t.Errorf("Expected number of locales \"6\", got %d", len(l1))
		t.Fail()
	}
	l2 := supportedLocales("en")
	if len(l2) != 1 {
		t.Errorf("Expected number of locales \"1\", got %d", len(l2))
		t.Fail()
	}
	l3 := supportedLocales("")
	if len(l3) != 0 {
		t.Errorf("Expected number of locales \"0\", got %d", len(l3))
		t.Fail()
	}
	l4 := ParseLocale("en_US")
	if l4.Lang != "en" || l4.Country != "US" {
		t.Errorf("Expected \"en\" and \"US\", got %s and %s", l4.Lang, l4.Country)
		t.Fail()
	}
	l5 := ParseLocale("en")
	if l5.Lang != "en" || l5.Country != "" {
		t.Errorf("Expected \"en\" and \"\", got %s and %s", l5.Lang, l5.Country)
		t.Fail()
	}

}
