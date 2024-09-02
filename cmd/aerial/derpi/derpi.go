package derpi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// DerpiResults is a struct to contain Derpibooru search results
type DerpiResults struct {
	Search []struct {
		ID               string        `json:"id"`
		CreatedAt        time.Time     `json:"created_at"`
		UpdatedAt        time.Time     `json:"updated_at"`
		DuplicateReports []interface{} `json:"duplicate_reports"`
		FirstSeenAt      time.Time     `json:"first_seen_at"`
		UploaderID       string        `json:"uploader_id"`
		Score            int           `json:"score"`
		CommentCount     int           `json:"comment_count"`
		Width            int           `json:"width"`
		Height           int           `json:"height"`
		FileName         string        `json:"file_name"`
		Description      string        `json:"description"`
		Uploader         string        `json:"uploader"`
		Image            string        `json:"image"`
		Upvotes          int           `json:"upvotes"`
		Downvotes        int           `json:"downvotes"`
		Faves            int           `json:"faves"`
		Tags             string        `json:"tags"`
		TagIds           []string      `json:"tag_ids"`
		AspectRatio      float64       `json:"aspect_ratio"`
		OriginalFormat   string        `json:"original_format"`
		MimeType         string        `json:"mime_type"`
		Sha512Hash       string        `json:"sha512_hash"`
		OrigSha512Hash   string        `json:"orig_sha512_hash"`
		SourceURL        string        `json:"source_url"`
		Representations  struct {
			ThumbTiny  string `json:"thumb_tiny"`
			ThumbSmall string `json:"thumb_small"`
			Thumb      string `json:"thumb"`
			Small      string `json:"small"`
			Medium     string `json:"medium"`
			Large      string `json:"large"`
			Tall       string `json:"tall"`
			Full       string `json:"full"`
		} `json:"representations"`
		IsRendered  bool `json:"is_rendered"`
		IsOptimized bool `json:"is_optimized"`
	} `json:"search"`
	Total        int           `json:"total"`
	Interactions []interface{} `json:"interactions"`
}

// Perform a Derpibooru search query with a given string of tags and an API key
func SearchDerpi(tags string) (DerpiResults, error) {

	// format for URL query
	tags += ",safe" // Enforce the safe tag for PG rating
	derpiTags := strings.Replace(tags, " ", "+", -1)

	// make URL query
	urlQuery := "https://derpibooru.org/search.json?q=" + derpiTags
	resp, err := http.Get(urlQuery)
	if err != nil {
		return DerpiResults{}, fmt.Errorf("Failed with HTTP error.")
	}

	// read response body
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return DerpiResults{}, fmt.Errorf("Failed with error reading response body.")
	}

	// parse json
	results := DerpiResults{}
	err = json.Unmarshal(respBody, &results)
	if err != nil {
		return DerpiResults{}, fmt.Errorf("Failed with JSON parsing error.")
	}

	return results, nil

}
