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
- `comicbd.ttf` is extracted with `7z e comic32.exe comicbd.ttf` from
  `comic32.exe` in the corefonts SourceForge mirror and embedded with
  `go:embed`. The user has confirmed permission to use it. `FONT.md` records
  the source URL and SHA-256.

## Rendering

`newRenderer() (*renderer, error)` parses the embedded font and overlays once.
`(*renderer).Render(oldName, newName string, newPic, old1, old2 image.Image) (image.Image, error)`
does no I/O. Steps:

1. Resize `newPic` to exactly 800x600 (Lanczos, aspect not preserved) via
   `disintegration/imaging`, then flatten it onto an opaque white background
   (`draw.Draw` with `image.White`, then the resized image with `draw.Over`).
   Transparent uploads (for example a PNG with an alpha channel) would
   otherwise render as a black rectangle, since the final JPEG has no alpha
   channel of its own; flattening onto white first keeps transparent areas
   looking like blank paper.
2. Resize `old1` to 172x259, draw `x1.png` over it at (0,0). Resize `old2` to
   221x242, draw `x2.png` over it at (0,0). Overlays are clipped to the photo
   bounds, matching `COMPOSITE_ATOP`. Both resized photos are flattened onto
   white the same way as `newPic` before the overlay is drawn.
3. Draw text. Font: Comic Sans MS Bold. Stroke: 1px `#006488`, drawn as the
   text offset in 8 directions under the fill. Names are trimmed and
   uppercased; the fixed words keep the original casing. The gradient title's
   clip mask is reset with `dc.ResetClip()` after drawing, since gg's `Pop`
   deliberately leaves the mask in place and it would otherwise leak into
   later text draws on the same context.

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
| `GET /f/{id}/image.jpg` | `302` to presigned URL for `result.jpg`, with a bounded `Cache-Control`.   |
| `GET /static/...`       | Embedded design-system CSS and fonts.                                      |
| `GET /treasure.gif`     | Embedded `assets/treasure.gif`.                                            |

Result page: "Bless Your New Friendship With <NEW>", the image, `treasure.gif`,
"Do you have another new friend..." link to `/`.

Validation on `POST /`:

- `http.MaxBytesReader` at 32 MB total; each file at most 10 MB.
- Both names required and non-empty after trimming, max 64 runes each, and
  every rune must be printable (`unicode.IsPrint`); this rejects control
  characters such as newlines or tabs while still allowing spaces.
- Each file decoded by content sniffing (`imaging.Decode` with
  `imaging.AutoOrientation(true)`, over GIF/JPEG/PNG registered via the
  standard library). Client MIME type is ignored. Auto-orientation applies
  the image's EXIF orientation tag, if present, so photos taken sideways or
  upside down on a phone come out upright.
- Each image is at most 40 megapixels (checked with `image.DecodeConfig`
  before decoding).
- `id` in `/f/{id}` must parse as a UUID, else 404.
- The four image objects upload in parallel; `meta.json` is written only
  after they all succeed, so a result page never points at a missing image.

Render concurrency: decoding, rendering, and JPEG-encoding an upload holds
several full-size pixel buffers in memory at once. `Server` holds a
`chan struct{}` semaphore with `maxConcurrentRenders` (2) slots; `POST /`
acquires a slot after names are validated and before reading uploads, and
releases it right after the JPEG is encoded, before the storage puts (the
decoded images are also dereferenced at that point so they can be
collected). Acquiring uses `select` against the request context, so a
client that disconnects while waiting gives up immediately instead of
holding the connection open; that case is logged at debug level and the
handler returns without further work.

## Config

Flags: `--bind` (default `:3000`), `--bucket` (default
`xe-friendship-ended`), `--base-url` (default `http://localhost:3000`, used
for absolute OpenGraph URLs), `--presign-expiry` (default `1h`). The S3
client comes from the existing `within.website/x/tigris.Client` helper,
which sets the Tigris endpoint and region. Credentials come from the
standard AWS environment variables (`AWS_ACCESS_KEY_ID` /
`AWS_SECRET_ACCESS_KEY`, or `AWS_PROFILE`). No secrets in the repo. For
local development, put `AWS_PROFILE=tigris` in `cmd/friendship-ended/.env`
and run the binary from that directory; `internal.HandleStartup()` loads
`.env` from the process's working directory, not the source tree.

`newServer(store Store, rend *renderer, baseURL string, presignExpiry time.Duration) *Server`
takes the presign expiry directly so the `image` handler can size its
`Cache-Control` header off of it (see HTTP above): `max-age` is
`min(300s, presignExpiry/2)` in whole seconds, never negative, so a cached
redirect can never outlive the presigned URL it points to.

The `http.Server` sets `ReadHeaderTimeout: 10s`, `ReadTimeout: 60s`,
`WriteTimeout: 90s`, and `IdleTimeout: 120s`. `main` listens for
`SIGINT`/`SIGTERM` via `signal.NotifyContext`, runs `ListenAndServe` in a
goroutine, and on signal calls `hs.Shutdown` with a 30s timeout context;
`http.ErrServerClosed` from `ListenAndServe` is treated as a normal
shutdown, not an error.

## Errors

- User errors: re-render the form with a message and status 400.
- Storage or render failures: log with `slog`, return 500.

## Testing

- `render_test.go`: table-driven. Output bounds are 800x600 for tiny (1x1),
  large, and GIF inputs. Empty and long names do not panic. A fully
  transparent new-friend picture renders as white, not black, in an area
  with no text or photo.
- `handlers_test.go`: table-driven against the in-memory store. Missing
  names, control characters in a name, non-image file, oversized upload,
  happy path (303 plus all five keys stored), unknown id (404), bad id
  (404), image redirect (302) with a `Cache-Control` bounded by the presign
  expiry, a storage failure that only fails one image key (meta.json must
  still not be stored), and the render semaphore fully released after a
  sequence of successful and failing requests.
- Golden smoke test writes `testdata/out.png` when `-update` is set, for
  visual inspection. `testdata/.gitignore` keeps it from being committed by
  accident.

## Out of scope

Gallery or listing, moderation, deletion, TTL/expiry, re-rendering from
stored sources.
