package mainsa

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	sunoLen  = 8 * time.Hour
	linjaLen = sunosInLinja * sunoLen
	sikeLen  = linjasInSike * linjaLen
	tawaLen  = sikesInTawa * sikeLen
	suliLen  = tawasInSuli * tawaLen
)

const (
	sunosInLinja = 3
	linjasInSike = 3
	sikesInTawa  = 3
	tawasInSuli  = 4

	sunosInSike = sunosInLinja * linjasInSike
	sunosInTawa = sunosInSike * sikesInTawa
	sunosInSuli = sunosInTawa * tawasInSuli
)

//go:generate stringer -type=Tawa

// Tawa is a set of 3 cycles, or a season.
type Tawa int

// The four tawas of a year.
const (
	Kasi Tawa = 1 + iota
	Seli
	Sin
	Lete
)

//go:generate stringer -type=Sike

// Sike is a set of 3 threads, or a month.
type Sike int

// The three sikes.
const (
	Kama Sike = 1 + iota
	Poka
	Monsi
)

//go:generate stringer -type=Nanpa

// Nanpa is a generic toki pona number
type Nanpa int

// Numbers
const (
	Wan Nanpa = 1 + iota
	Tu
	Mute
)

// TenpoNimi is the name of a given time. The remainder time is given as a time.Duration and should be shown in hour:minute format using time constant "3:04".
type TenpoNimi struct {
	Year      int   // suli
	Season    Tawa  // tawa
	Month     Sike  // sike
	Week      Nanpa // linja
	Day       Nanpa // suno
	Remainder time.Duration
}

func awen(d time.Duration) string {
	return fmt.Sprintf("%d:%02d", int(d.Hours()), int(d.Minutes())%60)
}

func (tn TenpoNimi) String() string {
	return strings.ToLower(fmt.Sprintf(
		"suli %d tawa %s sike %s linja %s suno %s awen %s",
		tn.Year,
		tn.Season,
		tn.Month,
		tn.Week,
		tn.Day,
		awen(tn.Remainder),
	))
}

const zeroDateUnix = 1538870400

// YearZero is the arbitrary anchor date from plane 432 to ma Insa year 1 planting season, coming cycle, first week, first day..
var YearZero = time.Unix(zeroDateUnix, 0)

// Errors
var (
	ErrBeforeEpoch = errors.New("mainsa: time before zero date")
)

// At returns the time in ma Insa for a given Go time.
func At(t time.Time) (TenpoNimi, error) {
	if t.Before(YearZero) {
		return TenpoNimi{}, ErrBeforeEpoch
	}

	dur := t.Sub(YearZero)

	var linjas = int(dur / linjaLen) // week => 3 days
	var sikes = int(dur / sikeLen)   // month => 3 weeks
	var tawas = int(dur / tawaLen)   // season => 3 months
	var sulis = int(dur / suliLen)   // year => 4 seasons
	sunosd := float64(dur) / float64(sunoLen)
	sunos := int(sunosd)
	rem := time.Duration((sunosd - float64(sunos)) * float64(sunoLen)).Round(time.Minute)

	result := TenpoNimi{
		Year:      sulis,
		Season:    Tawa(tawas%tawasInSuli) + 1,
		Month:     Sike(sikes%sikesInTawa) + 1,
		Week:      Nanpa(linjas%linjasInSike) + 1,
		Day:       Nanpa(sunos%sunosInLinja) + 1,
		Remainder: rem,
	}

	return result, nil
}
