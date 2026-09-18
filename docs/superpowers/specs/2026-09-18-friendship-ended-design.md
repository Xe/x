# friendship-ended design

Date: 2026-09-18

## Goal

Port the PHP "Friendship Ended Generator" (`var/friendship-ended`, by Leigh
Aucoin, GPL-2.0) to a Go binary at `cmd/friendship-ended`. Render images with
`github.com/fogleman/gg`, store renders and source uploads in Tigris, and style
the web UI with the xe-design-system.

## Behavior of the original

- Form: old friend name, new friend name, one new friend photo, two old friend
  photos (GIF/JPEG/PNG).
- Render: new friend photo stretched to 800x600. Old friend photos resized to
  172x259 and 221x242, overlaid with `x1.png` / `x2.png`, composited at
  (0,341) and (579,358). Text drawn on top.
- Result page: inline JPEG, `treasure.gif`, link back to the form. Nothing is
  stored.

## Layout

```text
cmd/friendship-ended/
  main.go        flags, S3 client, routes
  render.go      pure gg compositing
  render_test.go
  store.go       Tigris wrapper behind an interface
  handlers.go    HTTP handlers
  handlers_test.go
  web.templ      form, result, error pages (xe-design-system)
  assets/
    x1.png, x2.png, treasure.gif   from the original project
    LICENSE                        GPL-2.0 text for the above
    README.md                      attribution to Leigh Aucoin
    comicbd.ttf                    Comic Sans MS Bold
    FONT.md                        provenance of comicbd.ttf
```

Binaries use `internal.HandleStartup()` and do not call `flag.Parse()`.

## Assets

- `x1.png`, `x2.png`, `treasure.gif` are copied from the original and marked
  GPL-2.0 via `assets/LICENSE` and `assets/README.md`.
- `comicbd.ttf` is extracted with `cabextract` from `comic32.exe` in the
  corefonts SourceForge mirror and embedded with `go:embed`. The user has
  confirmed permission to use it. `FONT.md` records the source URL and
  SHA-256.

## Rendering

`newRenderer() (*renderer, error)` parses the embedded font and overlays once.
`(*renderer).Render(oldName, newName string, newPic, old1, old2 image.Image) (image.Image, error)`
does no I/O. Steps:

1. Resize `newPic` to exactly 800x600 (Lanczos, aspect not preserved) via
   `disintegration/imaging`.
2. Resize `old1` to 172x259, draw `x1.png` over it at (0,0). Resize `old2` to
   221x242, draw `x2.png` over it at (0,0). Overlays are clipped to the photo
   bounds, matching `COMPOSITE_ATOP`.
3. Draw text. Font: Comic Sans MS Bold. Stroke: 1px `#006488`, drawn as the
   text offset in 8 directions under the fill. Names are trimmed and
   uppercased; the fixed words keep the original casing.

   | Text                          | Size | Scale    | Position  | Fill                         |
   | ----------------------------- | ---- | -------- | --------- | ---------------------------- |
   | `Friendship ended with <OLD>` | 58   | (0.8, 2) | (0, 40)   | gradient `#CF4E09`-`#00B92C` |
   | `Now`                         | 38   | (1, 2)   | (300,70)  | `#DF0676`                    |
   | `<NEW>`                       | 38   | (1, 2)   | (300,110) | `#AB5955`                    |
   | `is my`                       | 38   | (1, 2)   | (300,150) | `#7C9535`                    |
   | `best friend`                 | 38   | (1, 2)   | (300,185) | `#4CBF1F`                    |

   Positions are in the pre-scale coordinate space, as in ImageMagick
   `annotation` after `scale`. The gradient is vertical over a 120px tall
   band, repeating like the original 50x120 pattern.

4. Composite old friend photos onto the canvas at (0,341) and (579,358).
5. Caller encodes as JPEG (quality 90).

## Storage (Tigris)

Bucket from `--bucket`. Keys under `friendships/<id>/`, where `<id>` is a
UUIDv7:

- `result.jpg`
- `new.<ext>`, `old1.<ext>`, `old2.<ext>`: original uploaded bytes, extension
  from the sniffed format
- `meta.json`: `{"old_name", "new_name", "created_at"}`

The bucket stays private. Access is via presigned GET URLs.

`Store` interface:

```go
type Store interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error) // ErrNotFound if missing
	Presign(ctx context.Context, key string) (string, error)
}
```

Implemented by an S3 client (aws-sdk-go-v2) and an in-memory fake for tests.

## HTTP

| Route                   | Behavior                                                                   |
| ----------------------- | -------------------------------------------------------------------------- |
| `GET /`                 | Form page.                                                                 |
| `POST /`                | Validate, render, store, `303` to `/f/<id>`.                               |
| `GET /f/{id}`           | Load `meta.json` (404 if missing), render result page with OpenGraph tags. |
| `GET /f/{id}/image.jpg` | `302` to presigned URL for `result.jpg`.                                   |
| `GET /static/...`       | Embedded design-system CSS and fonts.                                      |
| `GET /treasure.gif`     | Embedded `assets/treasure.gif`.                                            |

Result page: "Bless Your New Friendship With <NEW>", the image, `treasure.gif`,
"Do you have another new friend..." link to `/`.

Validation on `POST /`:

- `http.MaxBytesReader` at 32 MB total; each file at most 10 MB.
- Both names required and non-empty after trimming. Max 64 runes each.
- Each file decoded by content sniffing (`image.Decode` with GIF/JPEG/PNG
  registered). Client MIME type is ignored.
- Each image is at most 40 megapixels (checked with `image.DecodeConfig`
  before decoding).
- `id` in `/f/{id}` must parse as a UUID, else 404.
- The four image objects upload in parallel; `meta.json` is written only
  after they all succeed, so a result page never points at a missing image.

## Config

Flags: `--bind` (default `:3000`), `--bucket` (default `friendship-ended`),
`--base-url` (default `http://localhost:3000`, used for absolute OpenGraph
URLs), `--presign-expiry` (default `1h`). The S3 client comes from the
existing `within.website/x/tigris.Client` helper, which sets the Tigris
endpoint and region. Credentials come from the standard `AWS_ACCESS_KEY_ID`
and `AWS_SECRET_ACCESS_KEY` env vars. No secrets in the repo.

## Errors

- User errors: re-render the form with a message and status 400.
- Storage or render failures: log with `slog`, return 500.

## Testing

- `render_test.go`: table-driven. Output bounds are 800x600 for tiny (1x1),
  large, and GIF inputs. Empty and long names do not panic.
- `handlers_test.go`: table-driven against the in-memory store. Missing
  names, non-image file, oversized upload, happy path (303 plus all five
  keys stored), unknown id (404), bad id (404), image redirect (302).
- Golden smoke test writes `testdata/out.png` when `-update` is set, for
  visual inspection.

## Out of scope

Gallery or listing, moderation, deletion, TTL/expiry, re-rendering from
stored sources.
