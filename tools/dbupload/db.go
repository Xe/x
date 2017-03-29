package main

type UploadImage struct {
	Image struct {
		SourceURL string `json:"source_url"`
		Tags      string `json:"tag_list"`
		ImageURL  string `json:"image_url"`
	} `json:"image"`
}

type Image struct {
	ID               string        `json:"id"`
	IDNumber         int           `json:"id_number"`
	CreatedAt        string        `json:"created_at"`
	UpdatedAt        string        `json:"updated_at"`
	DuplicateReports []interface{} `json:"duplicate_reports"`
	FileName         string        `json:"file_name"`
	Description      string        `json:"description"`
	Uploader         string        `json:"uploader"`
	Image            string        `json:"image"`
	Score            int           `json:"score"`
	Upvotes          int           `json:"upvotes"`
	Downvotes        int           `json:"downvotes"`
	Faves            int           `json:"faves"`
	CommentCount     int           `json:"comment_count"`
	Tags             string        `json:"tags"`
	TagIds           []string      `json:"tag_ids"`
	Width            int           `json:"width"`
	Height           int           `json:"height"`
	AspectRatio      float64       `json:"aspect_ratio"`
	OriginalFormat   string        `json:"original_format"`
	MimeType         string        `json:"mime_type"`
	Sha512Hash       string        `json:"sha512_hash"`
	OrigSha512Hash   string        `json:"orig_sha512_hash"`
	SourceURL        string        `json:"source_url"`
	License          string        `json:"license"`
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
}
