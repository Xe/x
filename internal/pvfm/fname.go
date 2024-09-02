package pvfm

import (
	"fmt"
	"strings"
	"time"

	"within.website/x/internal/pvfm/pvl"
)

func GenFilename() (string, error) {
	cal, err := pvl.Get()
	if err != nil {
		return "", nil
	}

	now := cal.Result[0]

	localTime := time.Now()
	thentime := time.Unix(now.StartTime, 0)
	if thentime.Unix() < localTime.Unix() {
		// return fmt.Sprintf("%s - %s.mp3", now.Title, localTime.Format(time.RFC822)), nil
	}

	now.Title = strings.Replace(now.Title, "/", "-slash-", 0)

	return fmt.Sprintf("%s - %s.mp3", now.Title, localTime.Format(time.RFC822)), nil
}
