# gluamarkdown

Markdown processor for [gopher-lua](https://github.com/yuin/gopher-lua)

## Installation

```
go get github.com/kohkimakimoto/gluamarkdown
```

## API

### `markdown.dostring(text)`

Returns HTML string generated from the markdown text.

### `markdown.dofile(file)`

Returns HTML string generated from the markdown text file. If this function fails, it returns `nil`, plus a string describing the error.

## Usage

```go
package main

import (
    "github.com/yuin/gopher-lua"
    "github.com/kohkimakimoto/gluamarkdown"
)

func main() {
    L := lua.NewState()
    defer L.Close()

    L.PreloadModule("markdown", gluamarkdown.Loader)
    if err := L.DoString(`
local markdown = require("markdown")
local output = markdown.dostring([=[
# glua markdown

Markdown processor for gopher-lua

]=])

print(output)
-- you will get the following:
-- <h1>glua markdown</h1>
--
-- <p>Markdown processor for gopher-lua</p>
--
    `); err != nil {
        panic(err)
    }
}
```

## Author

Kohki Makimoto <kohki.makimoto@gmail.com>

## License

MIT license.
