package main

import (
	"embed"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/adrg/frontmatter"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	//go:embed docs
	docsFS embed.FS
)

type Doc struct {
	gorm.Model
	Frontmatter Frontmatter `gorm:"embedded"`
	Body        string
	BodyHTML    string
}

type Frontmatter struct {
	Title   string `yaml:"title"`
	Date    string `yaml:"date"`
	Updated string `yaml:"updated"`
	Slug    string `yaml:"slug" gorm:"uniqueIndex"`
}

func (s *Server) importDocs() error {
	files, err := docsFS.ReadDir("docs")
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fin, err := docsFS.Open("docs/" + file.Name())
		if err != nil {
			return err
		}
		defer fin.Close()

		doc := Doc{}

		rest, err := frontmatter.Parse(fin, &doc.Frontmatter)
		if err != nil {
			return err
		}

		extensions := parser.CommonExtensions | parser.AutoHeadingIDs
		p := parser.NewWithExtensions(extensions)

		htmlFlags := html.CommonFlags | html.HrefTargetBlank
		opts := html.RendererOptions{Flags: htmlFlags}
		renderer := html.NewRenderer(opts)

		pageHTML := markdown.ToHTML(rest, p, renderer)

		doc.Body = string(rest)
		doc.BodyHTML = string(pageHTML)

		if err := s.dao.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"body",
				"body_html",
				"date",
				"updated",
			}),
		}).Create(&doc).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) docsHandler(w http.ResponseWriter, r *http.Request) {
	var d Doc

	var navbar templ.Component

	tu, ok := s.getTelegramUserData(r)
	if ok {
		navbar = authedNavBar(tu)
	} else {
		navbar = anonNavBar(false)
	}

	slug := strings.TrimPrefix(r.URL.Path, "/docs")

	if err := s.dao.db.WithContext(r.Context()).Where("slug = ?", slug).First(&d).Error; err != nil {
		templ.Handler(
			base("Not found", nil, anonNavBar(false), notFoundPage()),
			templ.WithStatus(http.StatusNotFound),
		).ServeHTTP(w, r)
		return
	}

	templ.Handler(
		base(d.Frontmatter.Title, nil, navbar, docRender(d)),
	).ServeHTTP(w, r)
}
