package i18n

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// L represents Lingo bundle, containing map of all Ts by locale,
// as well as default locale and list of supported locales
type L struct {
	bundle    map[string]T
	deflt     string
	supported []Locale
}

func (l *L) exists(locale string) bool {
	_, exists := l.bundle[locale]
	return exists
}

// TranslationsForRequest will get the best matched T for given
// Request. If no T is found, returns default T
func (l *L) TranslationsForRequest(r *http.Request) T {
	locales := GetLocales(r)
	for _, locale := range locales {
		t, exists := l.bundle[locales[0].Name()]
		if exists {
			return t
		}
		for _, sup := range l.supported {
			if locale.Lang == sup.Lang {
				return l.bundle[sup.Name()]
			}
		}
	}
	return l.bundle[l.deflt]
}

// TranslationsForLocale will get the T for specific locale.
// If no locale is found, returns default T
func (l *L) TranslationsForLocale(locale string) T {
	t, exists := l.bundle[locale]
	if exists {
		return t
	}
	return l.bundle[l.deflt]
}

// T represents translations map for specific locale
type T struct {
	transl map[string]interface{}
}

// Value traverses the translations map and finds translation for
// given key. If no translation is found, returns value of given key.
func (t T) Value(key string, args ...string) string {
	if t.exists(key) {
		res, ok := t.transl[key].(string)
		if ok {
			return t.parseArgs(res, args)
		}
	}
	ksplt := strings.Split(key, ".")
	for i := range ksplt {
		k1 := strings.Join(ksplt[0:i], ".")
		k2 := strings.Join(ksplt[i:len(ksplt)], ".")
		if t.exists(k1) {
			newt := &T{
				transl: t.transl[k1].(map[string]interface{}),
			}
			return newt.Value(k2, args...)
		}
	}
	return key
}

// parseArgs replaces the argument placeholders with given arguments
func (t T) parseArgs(value string, args []string) string {
	res := value
	for i := 0; i < len(args); i++ {
		tok := "{" + strconv.Itoa(i) + "}"
		res = strings.Replace(res, tok, args[i], -1)
	}
	return res
}

// exists checks if value exists for given key
func (t T) exists(key string) bool {
	_, ok := t.transl[key]
	return ok
}

// New creates the Lingo bundle.
// Params:
// Default locale, to be used when requested locale
// is not found.
// Path, absolute or relative path to a folder where
// translation .json files are kept
func New(deflt, path string) *L {
	files, _ := ioutil.ReadDir(path)
	l := &L{
		bundle:    make(map[string]T),
		deflt:     deflt,
		supported: make([]Locale, 0),
	}
	for _, f := range files {
		fileName := f.Name()
		dat, err := ioutil.ReadFile(path + "/" + fileName)
		if err != nil {
			log.Printf("Cannot read file %s, file corrupt.", fileName)
			log.Printf("Error: %s", err)
			continue
		}
		t := T{
			transl: make(map[string]interface{}),
		}
		err = json.Unmarshal(dat, &t.transl)
		if err != nil {
			log.Printf("Cannot read file %s, invalid JSON.", fileName)
			log.Printf("Error: %s", err)
			continue
		}
		locale := strings.Split(fileName, ".")[0]
		l.supported = append(l.supported, ParseLocale(locale))
		l.bundle[locale] = t
	}
	return l
}
