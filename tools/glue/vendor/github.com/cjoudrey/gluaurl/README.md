# gluaurl

[![](https://travis-ci.org/cjoudrey/gluaurl.svg)](https://travis-ci.org/cjoudrey/gluaurl)

gluahttp provides an easy way to parse and build URLs from within [GopherLua](https://github.com/yuin/gopher-lua).

## Installation

```
go get github.com/cjoudrey/gluaurl
```

## Usage

```go
import "github.com/yuin/gopher-lua"
import "github.com/cjoudrey/gluaurl"

func main() {
    L := lua.NewState()
    defer L.Close()

    L.PreloadModule("url", gluaurl.Loader)

    if err := L.DoString(`

        local url = require("url")

        parsed_url = url.parse("http://example.com/")

        print(parsed_url.host)

    `); err != nil {
        panic(err)
    }
}
```

## API

- [`url.parse(url)`](#urlparseurl)
- [`url.build(options)`](#urlbuildoptions)
- [`url.build_query_string(query_params)`](#urlbuild_query_stringquery_params)
- [`url.resolve(from, to)`](#urlresolvefrom-to)

### url.parse(url)

Parse URL into a table of key/value components.

**Attributes**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| url     | String | URL to parsed |

**Returns**

Table with parsed URL or (nil, error message)

| Name     | Type   | Description |
| -------- | ------ | ----------- |
| scheme   | String | Scheme of the URL |
| username | String | Username |
| password | String | Password |
| host     | String | Host and port of the URL |
| path     | String | Path |
| query    | String | Query string |
| fragment | String | Fragment |

### url.build(options)

Assemble a URL string from a table of URL components.

**Attributes**

| Name    | Type  | Description |
| ------- | ----- | ----------- |
| options | Table | Table with URL components, see [`url.parse`](#urlparseurl) for list of valid components |

**Returns**

String

### url.build_query_string(query_params)

Assemble table of query string parameters into a string.

**Attributes**

| Name         | Type  | Description |
| ------------ | ----- | ----------- |
| query_params | Table | Table with query parameters |

**Returns**

String

### url.resolve(from, to)

Take a base URL, and a href URL, and resolve them as a browser would for an anchor tag.

| Name | Type   | Description |
| ---- | ------ | ----------- |
| from | String | base URL |
| to | String | href URL |

**Returns**

String or (nil, error message)
