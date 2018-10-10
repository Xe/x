package mainsa

import (
	"testing"
	"time"
)

func TestSuliLen(t *testing.T) {
	if suliLen != sunosInSuli*sunoLen {
		t.Fatalf("expected a year to be %d sunos, got: %d sunos", sunosInSuli, suliLen/sunoLen)
	}

	t.Logf("%d sunos in linja", sunosInLinja)
	t.Logf("%d sunos in sike", sunosInSike)
	t.Logf("%d sunos in tawa", sunosInTawa)
	t.Logf("%d sunos in suli", sunosInSuli)
}

func TestAt(t *testing.T) {
	cases := []struct {
		name    string
		inp     time.Time
		wantErr bool
		outp    TenpoNimi
	}{
		{
			name:    "before",
			inp:     YearZero.Add(-1 * time.Hour),
			wantErr: true,
		},
		{
			name: "second_day",
			inp:  YearZero.Add(sunoLen),
			outp: TenpoNimi{
				Year:      0,
				Season:    Kasi,
				Month:     Kama,
				Week:      Wan,
				Day:       Tu,
				Remainder: time.Duration(0),
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			tn, err := At(cs.inp)
			if err != nil && !cs.wantErr {
				t.Fatal(err)
			}

			if cs.wantErr && err == nil {
				t.Fatal("wanted an error but got none")
			}

			if cs.wantErr {
				return
			}

			cts := cs.outp.String()
			tts := tn.String()
			t.Logf("expected: %s", cts)
			t.Logf("got:      %s", tts)
			if cts != tts {
				t.Fatal("see -v")
			}
		})
	}
}
