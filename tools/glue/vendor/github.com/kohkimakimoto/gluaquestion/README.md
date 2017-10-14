# gluaquestion

A library of [gopher-lua](https://github.com/yuin/gopher-lua) to prompt the user for input.

## Installation

```
go get github.com/kohkimakimoto/gluaquestion
```

## API

* `question.ask(text)`
* `question.secret(text)`

## Usage

```go
package main

import (
    "github.com/yuin/gopher-lua"
    "github.com/kohkimakimoto/gluaquestion"
)

func main() {
    L := lua.NewState()
    defer L.Close()

    L.PreloadModule("question", gluaquestion.Loader)
    if err := L.DoString(`
local question = require("question")

local name = question.ask("What's your name: ")
print("hello " .. name)

-- What's your name: kohki
-- hello kohki

`); err != nil {
        panic(err)
    }
}
```
