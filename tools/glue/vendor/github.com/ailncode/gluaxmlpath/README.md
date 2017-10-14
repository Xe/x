# gluaxmlpath

gluaxmlpath provides an easy way to use [xmlpath](https://github.com/go-xmlpath/xmlpath) from within [GopherLua](https://github.com/yuin/gopher-lua).

## Installation

```
go get github.com/ailncode/gluaxmlpath
```

## Usage

```go
package main

import (
	"github.com/ailncode/gluaxmlpath"
	"github.com/yuin/gopher-lua"
)

func main() {
	L := lua.NewState()
	defer L.Close()

	gluaxmlpath.Preload(L)

	if err := L.DoString(`
        xml ="<bookist><book>x1</book><book>x2</book><book>x3</book></booklist>"
        local xmlpath = require("xmlpath")
        node,err = xmlpath.loadxml(xml)
        path,err = xmlpath.compile("//book")
        it = path:iter(node)
        for k,v in pairs(it) do
            print(k,v:string())
        end
    `); err != nil {
		panic(err)
	}
}
```
