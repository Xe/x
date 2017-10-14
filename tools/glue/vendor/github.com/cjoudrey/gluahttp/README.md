# gluahttp

[![](https://travis-ci.org/cjoudrey/gluahttp.svg)](https://travis-ci.org/cjoudrey/gluahttp)

gluahttp provides an easy way to make HTTP requests from within [GopherLua](https://github.com/yuin/gopher-lua).

## Installation

```
go get github.com/cjoudrey/gluahttp
```

## Usage

```go
import "github.com/yuin/gopher-lua"
import "github.com/cjoudrey/gluahttp"

func main() {
    L := lua.NewState()
    defer L.Close()

    L.PreloadModule("http", NewHttpModule(&http.Client{}).Loader)

    if err := L.DoString(`

        local http = require("http")

        response, error_message = http.request("GET", "http://example.com", {
            query="page=1"
            headers={
                Accept="*/*"
            }
        })

    `); err != nil {
        panic(err)
    }
}
```

## API

- [`http.delete(url [, options])`](#httpdeleteurl--options)
- [`http.get(url [, options])`](#httpgeturl--options)
- [`http.head(url [, options])`](#httpheadurl--options)
- [`http.patch(url [, options])`](#httppatchurl--options)
- [`http.post(url [, options])`](#httpposturl--options)
- [`http.put(url [, options])`](#httpputurl--options)
- [`http.request(method, url [, options])`](#httprequestmethod-url--options)
- [`http.request_batch(requests)`](#httprequest_batchrequests)
- [`http.response`](#httpresponse)

### http.delete(url [, options])

**Attributes**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| url     | String | URL of the resource to load |
| options | Table  | Additional options |

**Options**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| query   | String | URL encoded query params |
| cookies | Table  | Additional cookies to send with the request |
| headers | Table  | Additional headers to send with the request |

**Returns**

[http.response](#httpresponse) or (nil, error message)

### http.get(url [, options])

**Attributes**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| url     | String | URL of the resource to load |
| options | Table  | Additional options |

**Options**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| query   | String | URL encoded query params |
| cookies | Table  | Additional cookies to send with the request |
| headers | Table  | Additional headers to send with the request |

**Returns**

[http.response](#httpresponse) or (nil, error message)

### http.head(url [, options])

**Attributes**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| url     | String | URL of the resource to load |
| options | Table  | Additional options |

**Options**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| query   | String | URL encoded query params |
| cookies | Table  | Additional cookies to send with the request |
| headers | Table  | Additional headers to send with the request |

**Returns**

[http.response](#httpresponse) or (nil, error message)

### http.patch(url [, options])

**Attributes**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| url     | String | URL of the resource to load |
| options | Table  | Additional options |

**Options**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| query   | String | URL encoded query params |
| cookies | Table  | Additional cookies to send with the request |
| body    | String | Request body. |
| form    | String | Deprecated. URL encoded request body. This will also set the `Content-Type` header to `application/x-www-form-urlencoded` |
| headers | Table  | Additional headers to send with the request |

**Returns**

[http.response](#httpresponse) or (nil, error message)

### http.post(url [, options])

**Attributes**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| url     | String | URL of the resource to load |
| options | Table  | Additional options |

**Options**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| query   | String | URL encoded query params |
| cookies | Table  | Additional cookies to send with the request |
| body    | String | Request body. |
| form    | String | Deprecated. URL encoded request body. This will also set the `Content-Type` header to `application/x-www-form-urlencoded` |
| headers | Table  | Additional headers to send with the request |

**Returns**

[http.response](#httpresponse) or (nil, error message)

### http.put(url [, options])

**Attributes**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| url     | String | URL of the resource to load |
| options | Table  | Additional options |

**Options**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| query   | String | URL encoded query params |
| cookies | Table  | Additional cookies to send with the request |
| body    | String | Request body. |
| form    | String | Deprecated. URL encoded request body. This will also set the `Content-Type` header to `application/x-www-form-urlencoded` |
| headers | Table  | Additional headers to send with the request |

**Returns**

[http.response](#httpresponse) or (nil, error message)

### http.request(method, url [, options])

**Attributes**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| method  | String | The HTTP request method |
| url     | String | URL of the resource to load |
| options | Table  | Additional options |

**Options**

| Name    | Type   | Description |
| ------- | ------ | ----------- |
| query   | String | URL encoded query params |
| cookies | Table  | Additional cookies to send with the request |
| body    | String | Request body. |
| form    | String | Deprecated. URL encoded request body. This will also set the `Content-Type` header to `application/x-www-form-urlencoded` |
| headers | Table  | Additional headers to send with the request |

**Returns**

[http.response](#httpresponse) or (nil, error message)

### http.request_batch(requests)

**Attributes**

| Name     | Type  | Description |
| -------- | ----- | ----------- |
| requests | Table | A table of requests to send. Each request item is by itself a table containing [http.request](#httprequestmethod-url--options) parameters for the request |

**Returns**

[[http.response](#httpresponse)] or ([[http.response](#httpresponse)], [error message])

### http.response

The `http.response` table contains information about a completed HTTP request.

**Attributes**

| Name        | Type   | Description |
| ----------- | ------ | ----------- |
| body        | String | The HTTP response body |
| body_size   | Number | The size of the HTTP reponse body in bytes |
| headers     | Table  | The HTTP response headers |
| cookies     | Table  | The cookies sent by the server in the HTTP response |
| status_code | Number | The HTTP response status code |
| url         | String | The final URL the request ended pointing to after redirects |
