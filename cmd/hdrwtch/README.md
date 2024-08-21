# hdrwtch

hdrwtch is a tool that watches for changes in the `Last-Modified` header of a URL. You can use this to monitor the freshness of a web page, or to trigger an action when a page is updated.

For more information, [read the docs](https://hdrwtch.xeserv.us/docs/).

## Tech stack

The code is written in Go and uses the following libraries/tools:

- [Templ](https://templ.guide) for templating
- [telego](https://pkg.go.dev/github.com/mymmrac/telego) for Telegram integration
- [Gorm](https://gorm.io) for database access
- [gormlite](https://pkg.go.dev/github.com/ncruces/go-sqlite3/gormlite) for SQLite3 support (via WebAssembly and [Wazero](https://wazero.io/))
- [HTMX](https://htmx.org) for client-side interactivity
- [Tailwind CSS](https://tailwindcss.com) for styling
- [Tabler icons](https://tablericons.com) for icons
