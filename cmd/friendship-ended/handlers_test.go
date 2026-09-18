package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/image/bmp"
)

const testBaseURL = "https://friends.example"

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, solid(w, h, color.RGBA{0x80, 0x40, 0x20, 0xff})); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodeBMP(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := bmp.Encode(&buf, solid(4, 4, color.White)); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// encodeHugeGIF hand-builds a minimal GIF whose logical screen descriptor
// declares the given width and height. image.DecodeConfig for GIF only
// reads the 13-byte header (plus an optional global color table, which is
// omitted here), so this never needs real pixel data and stays tiny on
// disk even when it declares a huge image.
func encodeHugeGIF(t *testing.T, w, h int) []byte {
	t.Helper()
	if w > 0xffff || h > 0xffff {
		t.Fatalf("encodeHugeGIF: %dx%d does not fit in a GIF header", w, h)
	}
	buf := make([]byte, 13)
	copy(buf, "GIF87a")
	binary.LittleEndian.PutUint16(buf[6:8], uint16(w))
	binary.LittleEndian.PutUint16(buf[8:10], uint16(h))
	// buf[10] packed fields: no global color table.
	// buf[11] background color index: unused.
	// buf[12] pixel aspect ratio: unused.
	return buf
}

type upload struct {
	field string
	data  []byte
}

func multipartBody(t *testing.T, fields map[string]string, files []upload) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range files {
		fw, err := mw.CreateFormFile(f.field, f.field+".bin")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(f.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

func testServer(t *testing.T) (*Server, *memStore) {
	t.Helper()
	store := newMemStore()
	return newServer(store, testRenderer(t), testBaseURL), store
}

var locationRE = regexp.MustCompile(`^/f/([0-9a-f-]{36})$`)

func TestCreate(t *testing.T) {
	goodNames := map[string]string{"old-friend-name": "mudasir", "new-friend-name": "salman"}
	pic := encodePNG(t, 64, 48)
	goodFiles := []upload{{"new-friend-pic", pic}, {"old-friend-1", pic}, {"old-friend-2", pic}}

	withFile := func(field string, data []byte) []upload {
		out := []upload{}
		for _, f := range goodFiles {
			if f.field == field {
				if data == nil {
					continue
				}
				f.data = data
			}
			out = append(out, f)
		}
		return out
	}

	for _, tt := range []struct {
		name       string
		fields     map[string]string
		files      []upload
		failPut    bool
		wantStatus int
		wantBody   string
	}{
		{name: "happy path", fields: goodNames, files: goodFiles, wantStatus: http.StatusSeeOther},
		{
			name:       "missing old name",
			fields:     map[string]string{"new-friend-name": "salman"},
			files:      goodFiles,
			wantStatus: http.StatusBadRequest,
			wantBody:   "old friend needs a name",
		},
		{
			name:       "blank new name",
			fields:     map[string]string{"old-friend-name": "mudasir", "new-friend-name": "   "},
			files:      goodFiles,
			wantStatus: http.StatusBadRequest,
			wantBody:   "new friend needs a name",
		},
		{
			name:       "name too long",
			fields:     map[string]string{"old-friend-name": strings.Repeat("a", 65), "new-friend-name": "salman"},
			files:      goodFiles,
			wantStatus: http.StatusBadRequest,
			wantBody:   "old friend needs a name",
		},
		{
			name:       "missing file",
			fields:     goodNames,
			files:      withFile("old-friend-2", nil),
			wantStatus: http.StatusBadRequest,
			wantBody:   "please attach a picture",
		},
		{
			name:       "not an image",
			fields:     goodNames,
			files:      withFile("old-friend-1", []byte("hello, this is text")),
			wantStatus: http.StatusBadRequest,
			wantBody:   "isn&#39;t a GIF, JPEG, or PNG",
		},
		{
			name:       "bmp is rejected",
			fields:     goodNames,
			files:      withFile("new-friend-pic", encodeBMP(t)),
			wantStatus: http.StatusBadRequest,
			wantBody:   "isn&#39;t a GIF, JPEG, or PNG",
		},
		{
			name:       "too many pixels",
			fields:     goodNames,
			files:      withFile("new-friend-pic", encodeHugeGIF(t, 7000, 7000)),
			wantStatus: http.StatusBadRequest,
			wantBody:   "too many pixels",
		},
		{
			name:       "file too big",
			fields:     goodNames,
			files:      withFile("new-friend-pic", make([]byte, maxFileSize+1)),
			wantStatus: http.StatusBadRequest,
			wantBody:   "bigger than 10 MB",
		},
		{
			name:       "storage failure",
			fields:     goodNames,
			files:      goodFiles,
			failPut:    true,
			wantStatus: http.StatusInternalServerError,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv, store := testServer(t)
			store.failPut = tt.failPut

			body, ct := multipartBody(t, tt.fields, tt.files)
			req := httptest.NewRequest(http.MethodPost, "/", body)
			req.Header.Set("Content-Type", ct)
			rec := httptest.NewRecorder()

			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body does not contain %q:\n%s", tt.wantBody, rec.Body.String())
			}
			if tt.wantStatus != http.StatusSeeOther {
				if len(store.objects) != 0 {
					t.Fatalf("stored %d objects on failure, want 0", len(store.objects))
				}
				return
			}

			m := locationRE.FindStringSubmatch(rec.Header().Get("Location"))
			if m == nil {
				t.Fatalf("Location = %q, want /f/<uuid>", rec.Header().Get("Location"))
			}
			prefix := "friendships/" + m[1] + "/"

			for key, wantType := range map[string]string{
				"result.jpg": "image/jpeg",
				"new.png":    "image/png",
				"old1.png":   "image/png",
				"old2.png":   "image/png",
				"meta.json":  "application/json",
			} {
				obj, ok := store.objects[prefix+key]
				if !ok {
					t.Errorf("missing object %s", prefix+key)
					continue
				}
				if obj.contentType != wantType {
					t.Errorf("%s content type = %q, want %q", key, obj.contentType, wantType)
				}
			}
			if len(store.objects) != 5 {
				t.Errorf("stored %d objects, want 5", len(store.objects))
			}

			var got meta
			if err := json.Unmarshal(store.objects[prefix+"meta.json"].data, &got); err != nil {
				t.Fatal(err)
			}
			if got.OldName != "mudasir" || got.NewName != "salman" || got.CreatedAt.IsZero() {
				t.Errorf("meta = %+v", got)
			}
		})
	}
}

func TestRead(t *testing.T) {
	const id = "0192f0c1-8a2b-7c3d-9e4f-5a6b7c8d9e0f"
	const unknown = "0192f0c1-8a2b-7c3d-9e4f-000000000000"

	for _, tt := range []struct {
		name         string
		path         string
		wantStatus   int
		wantBody     []string
		wantLocation string
		wantType     string
	}{
		{name: "form", path: "/", wantStatus: http.StatusOK, wantBody: []string{`name="old-friend-name"`, "FRIEND"}},
		{
			name:       "result page",
			path:       "/f/" + id,
			wantStatus: http.StatusOK,
			wantBody: []string{
				"SALMAN",
				`content="` + testBaseURL + "/f/" + id + `/image.jpg"`,
				`src="/f/` + id + `/image.jpg"`,
			},
		},
		{name: "unknown id", path: "/f/" + unknown, wantStatus: http.StatusNotFound},
		{name: "bad id", path: "/f/not-a-uuid", wantStatus: http.StatusNotFound},
		{
			name:         "image redirect",
			path:         "/f/" + id + "/image.jpg",
			wantStatus:   http.StatusFound,
			wantLocation: "https://tigris.example/friendships/" + id + "/result.jpg?X-Amz-Signature=fake",
		},
		{name: "image bad id", path: "/f/nope/image.jpg", wantStatus: http.StatusNotFound},
		{name: "treasure", path: "/treasure.gif", wantStatus: http.StatusOK, wantType: "image/gif"},
		{name: "design css", path: "/static/colors_and_type.css", wantStatus: http.StatusOK, wantType: "text/css; charset=utf-8"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv, store := testServer(t)
			metaJSON, err := json.Marshal(meta{OldName: "mudasir", NewName: "salman", CreatedAt: time.Now()})
			if err != nil {
				t.Fatal(err)
			}
			store.objects["friendships/"+id+"/meta.json"] = memObject{contentType: "application/json", data: metaJSON}

			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			for _, want := range tt.wantBody {
				if !strings.Contains(rec.Body.String(), want) {
					t.Errorf("body does not contain %q", want)
				}
			}
			if tt.wantLocation != "" && rec.Header().Get("Location") != tt.wantLocation {
				t.Errorf("Location = %q, want %q", rec.Header().Get("Location"), tt.wantLocation)
			}
			if tt.wantType != "" && rec.Header().Get("Content-Type") != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", rec.Header().Get("Content-Type"), tt.wantType)
			}
		})
	}
}
