package models

import (
	"context"
	"time"

	"gorm.io/gorm"
	"within.website/x/proto/external/jsonfeed"
)

// Blogpost is a single blogpost from a JSON Feed.
//
// This is tracked in the database so that the Announcer service can avoid double-posting.
type Blogpost struct {
	gorm.Model            // adds CreatedAt, UpdatedAt, DeletedAt
	ID          string    `gorm:"uniqueIndex"` // unique identifier for the blogpost
	URL         string    `gorm:"uniqueIndex"` // URL of the blogpost
	Title       string    // title of the blogpost
	BodyHTML    string    // HTML body of the blogpost
	Image       string    // URL of the image for the blogpost
	PublishedAt time.Time // when the blogpost was published
}

func (d *DAO) HasBlogpost(ctx context.Context, postURL string) (bool, error) {
	var count int64
	if err := d.db.WithContext(ctx).Model(&Blogpost{}).Where("url = ?", postURL).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (d *DAO) InsertBlogpost(ctx context.Context, post *jsonfeed.Item) (*Blogpost, error) {
	bp := &Blogpost{
		ID:          post.GetId(),
		URL:         post.GetUrl(),
		Title:       post.GetTitle(),
		BodyHTML:    post.GetContentHtml(),
		PublishedAt: post.GetDatePublished().AsTime(),
	}

	if err := d.db.WithContext(ctx).Create(bp).Error; err != nil {
		return nil, err
	}

	return bp, nil
}
