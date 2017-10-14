# gluafs

filesystem utility for [gopher-lua](https://github.com/yuin/gopher-lua). This project is inspired by [layeh/gopher-lfs](https://github.com/layeh/gopher-lfs).

## Installation

```
go get github.com/kohkimakimoto/gluafs
```

## API

### `fs.exists(file)`

Returns true if the file exists.

### `fs.read(file)`

Reads file content and return it. If this function fails, it returns `nil`, plus a string describing the error.

### `fs.write(file, content, [mode])`

Writes content to the file. If this function fails, it returns `nil`, plus a string describing the error.

### `fs.mkdir(path, [mode, recursive])`

Create directory. If this function fails, it returns `nil`, plus a string describing the error.

### `fs.remove(path, [recursive])`

Remove path. If this function fails, it returns `nil`, plus a string describing the error.

### `fs.symlink(target, link)`

Create symbolic link. If this function fails, it returns `nil`, plus a string describing the error.

### `fs.dirname(path)`

Returns all but the last element of path.

### `fs.basename(path)`

Returns the last element of path.

### `fs.realpath(path)`

Returns the real path of a given path in the os. If this function fails, it returns `nil`, plus a string describing the error.

### `fs.getcwd()`

Returns the current working directory. If this function fails, it returns `nil`, plus a string describing the error.

### `fs.chdir(path)`

Changes the current working directory. If this function fails, it returns `nil`, plus a string describing the error.

### `fs.file()`

Returns the script file path. If this function fails, it returns `nil`, plus a string describing the error.

### `fs.dir()`

Returns the directory path that is parent of the script file. If this function fails, it returns `nil`, plus a string describing the error.

### `fs.glob(pattern, function)`

Run the callback function with the files matching pattern. See below example:

```lua
local fs = require("fs")
local ret, err = fs.glob("/tmp/*", function(file)
	print(file.path)
	print(file.realpath)
end)
```

## Usage

```go
package main

import (
    "github.com/yuin/gopher-lua"
    "github.com/kohkimakimoto/gluafs"
)

func main() {
    L := lua.NewState()
    defer L.Close()

    L.PreloadModule("fs", gluafs.Loader)
    if err := L.DoString(`
local fs = require("fs")
local ret = fs.exists("path/to/file")

`); err != nil {
        panic(err)
    }
}
```

## Author

Kohki Makimoto <kohki.makimoto@gmail.com>

## License

MIT license.
