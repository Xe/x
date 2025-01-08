package ytdlp

import (
	"bytes"
	"context"
	"database/sql/driver"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"within.website/x/internal"
)

const dateFormat = "20060102"

type VideoMetadata struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Thumbnail      string `json:"thumbnail"`
	DurationString string `json:"duration_string"`
	UploadDate     Date   `json:"upload_date"`
	URL            string `json:"url" gorm:"uniqueIndex"`
}

type Date struct {
	time.Time
}

func (d *Date) MarshalJSON() ([]byte, error) {
	result := d.Format(dateFormat)

	return []byte(fmt.Sprintf("%q", result)), nil
}

func (d *Date) UnmarshalJSON(data []byte) error {
	str := string(data)
	str = str[1 : len(str)-1]

	parsedTime, err := time.Parse(dateFormat, str)
	if err != nil {
		return err
	}

	d.Time = parsedTime
	return nil
}

// Value implements the driver.Valuer interface
func (d Date) Value() (driver.Value, error) {
	return d.Format(dateFormat), nil
}

// Scan implements the sql.Scanner interface
func (d *Date) Scan(value interface{}) error {
	if value == nil {
		*d = Date{Time: time.Time{}}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*d = Date{Time: v}
	case []byte:
		parsedTime, err := time.Parse(dateFormat, string(v))
		if err != nil {
			return err
		}
		*d = Date{Time: parsedTime}
	case string:
		parsedTime, err := time.Parse(dateFormat, v)
		if err != nil {
			return err
		}
		*d = Date{Time: parsedTime}
	default:
		return fmt.Errorf("cannot scan type %T into Date", value)
	}

	return nil
}

func Metadata(ctx context.Context, url string) (*VideoMetadata, error) {
	result, err := internal.RunJSON[VideoMetadata](ctx, "yt-dlp", "--dump-json", url)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func Download(ctx context.Context, url, to string) error {
	exePath, err := exec.LookPath("yt-dlp")
	if err != nil {
		return fmt.Errorf("can't find yt-dlp: %w", err)
	}

	cmd := exec.CommandContext(ctx, exePath, "-o", filepath.Join(to, "%(id)s.%(ext)s"), "--write-info-json", url)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
	cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)

	if err := cmd.Run(); err != nil {
		// TODO(Xe): return stdout/err?
		return fmt.Errorf("can't download %s: %w", url, err)
	}

	return nil
}
