package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const (
	maxUploadSize  = 32 << 20
	maxFileSize    = 10 << 20
	maxImagePixels = 40_000_000
	maxNameRunes   = 64
)

// User-facing upload errors. Their text is shown on the form.
var (
	errNoFile        = errors.New("please attach a picture")
	errFileTooBig    = errors.New("that picture is bigger than 10 MB")
	errNotImage      = errors.New("that file isn't a GIF, JPEG, or PNG")
	errImageTooLarge = errors.New("that picture has too many pixels")
)

// meta is stored as meta.json next to each render.
type meta struct {
	OldName   string    `json:"old_name"`
	NewName   string    `json:"new_name"`
	CreatedAt time.Time `json:"created_at"`
}

// Server serves the friendship ended web app.
type Server struct {
	store    Store
	renderer *renderer
	baseURL  string
}

func newServer(store Store, rend *renderer, baseURL string) *Server {
	return &Server{
		store:    store,
		renderer: rend,
		baseURL:  strings.TrimSuffix(baseURL, "/"),
	}
}

// Handler returns the HTTP routes for the app.
func (s *Server) Handler() http.Handler {
	static, err := fs.Sub(content, "static")
	if err != nil {
		panic(fmt.Sprintf("static files missing from embed: %v", err))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("POST /{$}", s.create)
	mux.HandleFunc("GET /f/{id}", s.show)
	mux.HandleFunc("GET /f/{id}/image.jpg", s.image)
	mux.HandleFunc("GET /treasure.gif", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, content, "assets/treasure.gif")
	})
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))

	return mux
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	templ.Handler(indexPage(formState{})).ServeHTTP(w, r)
}

func (s *Server) formError(w http.ResponseWriter, r *http.Request, f formState) {
	templ.Handler(indexPage(f), templ.WithStatus(http.StatusBadRequest)).ServeHTTP(w, r)
}

func (s *Server) internalError(w http.ResponseWriter, r *http.Request, msg string, err error) {
	slog.ErrorContext(r.Context(), msg, "err", err)
	templ.Handler(
		errorPage("Something broke", "Something went wrong on our end. Your friendship is still over, though."),
		templ.WithStatus(http.StatusInternalServerError),
	).ServeHTTP(w, r)
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	templ.Handler(
		errorPage("Friendship not found", "That friendship either never existed or was never written down."),
		templ.WithStatus(http.StatusNotFound),
	).ServeHTTP(w, r)
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		s.formError(w, r, formState{Error: "That upload was too big or broken. Each picture can be at most 10 MB."})
		return
	}
	defer r.MultipartForm.RemoveAll()

	f := formState{
		OldName: strings.TrimSpace(r.FormValue("old-friend-name")),
		NewName: strings.TrimSpace(r.FormValue("new-friend-name")),
	}

	if !validName(f.OldName) {
		f.Error = fmt.Sprintf("Your old friend needs a name, %d characters or less.", maxNameRunes)
		s.formError(w, r, f)
		return
	}

	if !validName(f.NewName) {
		f.Error = fmt.Sprintf("Your new friend needs a name, %d characters or less.", maxNameRunes)
		s.formError(w, r, f)
		return
	}

	uploads := map[string]*imageUpload{}
	for _, field := range []struct{ name, label string }{
		{"new-friend-pic", "New friend picture"},
		{"old-friend-1", "First old friend picture"},
		{"old-friend-2", "Second old friend picture"},
	} {
		up, err := readUpload(r, field.name)
		if err != nil {
			f.Error = fmt.Sprintf("%s: %v.", field.label, err)
			s.formError(w, r, f)
			return
		}
		uploads[field.name] = up
	}

	img, err := s.renderer.Render(f.OldName, f.NewName,
		uploads["new-friend-pic"].img, uploads["old-friend-1"].img, uploads["old-friend-2"].img)
	if err != nil {
		s.internalError(w, r, "can't render image", err)
		return
	}

	var result bytes.Buffer
	if err := jpeg.Encode(&result, img, &jpeg.Options{Quality: 90}); err != nil {
		s.internalError(w, r, "can't encode jpeg", err)
		return
	}

	id, err := uuid.NewV7()
	if err != nil {
		s.internalError(w, r, "can't make id", err)
		return
	}
	prefix := "friendships/" + id.String() + "/"

	objects := []struct {
		key, contentType string
		data             []byte
	}{
		{prefix + "result.jpg", "image/jpeg", result.Bytes()},
		{prefix + "new." + uploads["new-friend-pic"].ext, uploads["new-friend-pic"].contentType, uploads["new-friend-pic"].data},
		{prefix + "old1." + uploads["old-friend-1"].ext, uploads["old-friend-1"].contentType, uploads["old-friend-1"].data},
		{prefix + "old2." + uploads["old-friend-2"].ext, uploads["old-friend-2"].contentType, uploads["old-friend-2"].data},
	}

	g, gctx := errgroup.WithContext(r.Context())
	for _, obj := range objects {
		g.Go(func() error {
			return s.store.Put(gctx, obj.key, obj.contentType, obj.data)
		})
	}
	if err := g.Wait(); err != nil {
		s.internalError(w, r, "can't store images", err)
		return
	}

	// meta.json goes last so a result page never points at missing images.
	metaJSON, err := json.Marshal(meta{OldName: f.OldName, NewName: f.NewName, CreatedAt: time.Now().UTC()})
	if err != nil {
		s.internalError(w, r, "can't marshal metadata", err)
		return
	}
	if err := s.store.Put(r.Context(), prefix+"meta.json", "application/json", metaJSON); err != nil {
		s.internalError(w, r, "can't store metadata", err)
		return
	}

	slog.InfoContext(r.Context(), "friendship ended", "id", id.String())
	http.Redirect(w, r, "/f/"+id.String(), http.StatusSeeOther)
}

func validName(name string) bool {
	n := utf8.RuneCountInString(name)
	return n > 0 && n <= maxNameRunes
}

type imageUpload struct {
	img         image.Image
	data        []byte
	ext         string
	contentType string
}

// readUpload reads and decodes one uploaded picture. Errors it returns are
// safe to show to the user.
func readUpload(r *http.Request, field string) (*imageUpload, error) {
	fin, hdr, err := r.FormFile(field)
	if err != nil {
		return nil, errNoFile
	}
	defer fin.Close()

	if hdr.Size > maxFileSize {
		return nil, errFileTooBig
	}

	data, err := io.ReadAll(io.LimitReader(fin, maxFileSize+1))
	if err != nil {
		return nil, errNotImage
	}
	if len(data) > maxFileSize {
		return nil, errFileTooBig
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errNotImage
	}

	var ext string
	switch format {
	case "gif", "png":
		ext = format
	case "jpeg":
		ext = "jpg"
	default:
		return nil, errNotImage
	}

	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxImagePixels {
		return nil, errImageTooLarge
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errNotImage
	}

	return &imageUpload{img: img, data: data, ext: ext, contentType: "image/" + format}, nil
}

func parseID(r *http.Request) (string, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return "", false
	}
	return id.String(), true
}

func (s *Server) show(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		s.notFound(w, r)
		return
	}

	data, err := s.store.Get(r.Context(), "friendships/"+id+"/meta.json")
	if errors.Is(err, ErrNotFound) {
		s.notFound(w, r)
		return
	}
	if err != nil {
		s.internalError(w, r, "can't load metadata", err)
		return
	}

	var m meta
	if err := json.Unmarshal(data, &m); err != nil {
		s.internalError(w, r, "can't parse metadata", err)
		return
	}

	templ.Handler(resultPage(s.baseURL, id, m)).ServeHTTP(w, r)
}

func (s *Server) image(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		s.notFound(w, r)
		return
	}

	u, err := s.store.Presign(r.Context(), "friendships/"+id+"/result.jpg")
	if err != nil {
		s.internalError(w, r, "can't presign image", err)
		return
	}

	w.Header().Set("Cache-Control", "private, max-age=300")
	http.Redirect(w, r, u, http.StatusFound)
}
