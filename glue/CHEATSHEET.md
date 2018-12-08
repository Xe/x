## `json`

```lua
local json = require "json"
```

Json encoder/decoder

The following functions are exposed by the library:

    decode(string): Decodes a JSON string. Returns nil and an error string if
                    the string could not be decoded.
    encode(value):  Encodes a value into a JSON string. Returns nil and an error
                    string if the value could not be encoded.

## `xmlpath`

```lua
local xmlpath = require "xmlpath"
```

XMLPath style iteration

    xml ="<bookist><book>x1</book><book>x2</book><book>x3</book></booklist>"
    local xmlpath = require("xmlpath")
    node,err = xmlpath.loadxml(xml)
    path,err = xmlpath.compile("//book")

    it = path:iter(node)
    for k,v in pairs(it) do
      print(k,v:string())
    end

## `http`

```lua
local http = require("http")
```

HTTP client library

### API

- [`http.delete(url [, options])`](#httpdeleteurl--options)
- [`http.get(url [, options])`](#httpgeturl--options)
- [`http.head(url [, options])`](#httpheadurl--options)
- [`http.patch(url [, options])`](#httppatchurl--options)
- [`http.post(url [, options])`](#httpposturl--options)
- [`http.put(url [, options])`](#httpputurl--options)
- [`http.request(method, url [, options])`](#httprequestmethod-url--options)
- [`http.request_batch(requests)`](#httprequest_batchrequests)
- [`http.response`](#httpresponse)

#### http.delete(url [, options])

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

#### http.get(url [, options])

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

#### http.head(url [, options])

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

#### http.patch(url [, options])

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

#### http.post(url [, options])

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

#### http.put(url [, options])

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

#### http.request(method, url [, options])

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

#### http.request_batch(requests)

**Attributes**

| Name     | Type  | Description |
| -------- | ----- | ----------- |
| requests | Table | A table of requests to send. Each request item is by itself a table containing [http.request](#httprequestmethod-url--options) parameters for the request |

**Returns**

[[http.response](#httpresponse)] or ([[http.response](#httpresponse)], [error message])

#### http.response

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

## `url`

```lua
local url = require "url"
```

URL parsing library

### API

- [`url.parse(url)`](#urlparseurl)
- [`url.build(options)`](#urlbuildoptions)
- [`url.build_query_string(query_params)`](#urlbuild_query_stringquery_params)
- [`url.resolve(from, to)`](#urlresolvefrom-to)

#### url.parse(url)

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

#### url.build(options)

Assemble a URL string from a table of URL components.

**Attributes**

| Name    | Type  | Description |
| ------- | ----- | ----------- |
| options | Table | Table with URL components, see [`url.parse`](#urlparseurl) for list of valid components |

**Returns**

String

#### url.build_query_string(query_params)

Assemble table of query string parameters into a string.

**Attributes**

| Name         | Type  | Description |
| ------------ | ----- | ----------- |
| query_params | Table | Table with query parameters |

**Returns**

String

#### url.resolve(from, to)

Take a base URL, and a href URL, and resolve them as a browser would for an anchor tag.

| Name | Type   | Description |
| ---- | ------ | ----------- |
| from | String | base URL |
| to | String | href URL |

**Returns**

String or (nil, error message)

## `env`

```lua
local env = require "env"
```

Environment manipulation

### API

#### `env.set(key, value)`

Same `os.setenv`

#### `env.get(key)`

Same `os.getenv`

#### `env.loadfile(file)`

Loads environment variables from a file. The file is as the following:

```
AAA=BBB
CCC=DDD
```

If this function fails, it returns `nil`, plus a string describing the error.

## `fs`

```lua
local fs = require "fs"
```

Filesystem manipulation

### API

#### `fs.exists(file)`

Returns true if the file exists.

#### `fs.read(file)`

Reads file content and return it. If this function fails, it returns `nil`, plus a string describing the error.

#### `fs.write(file, content, [mode])`

Writes content to the file. If this function fails, it returns `nil`, plus a string describing the error.

#### `fs.mkdir(path, [mode, recursive])`

Create directory. If this function fails, it returns `nil`, plus a string describing the error.

#### `fs.remove(path, [recursive])`

Remove path. If this function fails, it returns `nil`, plus a string describing the error.

#### `fs.symlink(target, link)`

Create symbolic link. If this function fails, it returns `nil`, plus a string describing the error.

#### `fs.dirname(path)`

Returns all but the last element of path.

#### `fs.basename(path)`

Returns the last element of path.

#### `fs.realpath(path)`

Returns the real path of a given path in the os. If this function fails, it returns `nil`, plus a string describing the error.

#### `fs.getcwd()`

Returns the current working directory. If this function fails, it returns `nil`, plus a string describing the error.

#### `fs.chdir(path)`

Changes the current working directory. If this function fails, it returns `nil`, plus a string describing the error.

#### `fs.file()`

Returns the script file path. If this function fails, it returns `nil`, plus a string describing the error.

#### `fs.dir()`

Returns the directory path that is parent of the script file. If this function fails, it returns `nil`, plus a string describing the error.

#### `fs.glob(pattern, function)`

Run the callback function with the files matching pattern. See below example:

```lua
local fs = require("fs")
local ret, err = fs.glob("/tmp/*", function(file)
  print(file.path)
  print(file.realpath)
end)
```

## `markdown`

```lua
local markdown = require "markdown"
```

Markdown -> HTML for string and file

### API

#### `markdown.dostring(text)`

Returns HTML string generated from the markdown text.

#### `markdown.dofile(file)`

Returns HTML string generated from the markdown text file. If this function fails, it returns `nil`, plus a string describing the error.

## `question`

```lua
local question = require "question"
```

Prompt library

### API

* `question.ask(text)`
* `question.secret(text)`

## `ssh`

```lua
local ssh = require "ssh"
```

SSH client library

https://github.com/kohkimakimoto/gluassh/blob/master/gluassh_test.go

## `template`

```lua
local template = require "template"
```

Go text templates

### API

#### `template.dostring(text, table)`

Returns string generated by text template with the table values. If this function fails, it returns `nil`, plus a string describing the error.

#### `template.dofile(file, table)`

Returns string generated by file template with the table values. If this function fails, it returns `nil`, plus a string describing the error.

## `yaml`

```lua
local yaml = require "yaml"
```

Yaml -> table parser

### API

#### `yaml.parse(string)`

Parses yaml formatted string and returns a table. If this function fails, it returns `nil`, plus a string describing the error.

## `flag`

```lua
local flag = require "flag"
```

Command line flag parsing.

See the tests here: https://github.com/otm/gluaflag

```lua
local flag = require "flag"

fs = flag.new()

fs:string("name", "foo", "String help string")
fs:intArg("title", 1, "Title")
fs:numberArg("title", 1, "Title")

flags = fs:parse(arg) -- arg is the remaining command line arguments
assert(flags.title == 2, "expected title to be 2")
assert(flags.title == 2.32, "expected title to be 2.32")
```

## `sh`

```lua
local sh = require "sh"
```

gluash is a interface to call any program as it were a function. Programs are executed asynchronously to enable streaming of data in pipes.

In all discussions bellow the imported module will be referred to as `sh`.

Commands are called just like functions, executed on the sh module.

```lua
sh.ls("/")
```

For commands that have exotic names, names that are reserved words, or to execute absolute or relative paths call the sh module directly.

```lua
sh("/bin/ls", "/")
```

#### Multiple Arguments
Commands with multiple arguments have to be invoked with a separate string for each argument.

```lua
-- this works
sh.ls("-la", "/")

-- this does not work
sh.ls("-la /")
```

#### Piping
Piping in sh is done almost like piping in the shell. Just call next command as a method on the previous command.

```lua
sh.du("-sb"):sort("-rn"):print()
```

If the command has a exotic name, or a reserved word, call the command through `cmd(path, ...args)`. The first argument in `cmd` is the path.

```lua
sh.du("-sb"):cmd("sort", "-rn"):print()
```

### Waiting for Processes
All commands are executed by default in the background, so one have to explicitly wait for a process to finish. There are several ways to wait for the command to finish.

* `print()`     - write stdout and stderr to stdout.
* `ok()`        - aborts execution if the command's exit code is not zero
* `success()`   - returns true of the commands exit code is zero
* `exitcode()`  - returns the exit code of the command

### Abort by Default
It is possible to set the module to abort on errors without checking. It can be practical in some occasions, however performance will be degraded. When global exit code checks are done the commands are run in series, even in pipes, and output is saved in memory buffers.

To enable global exit code settings call the sh module with an table with the key `abort` set to true.

```lua
sh{abort=true}
```

To read current settings in the module call the module with an empty table.
```lua
configuration = sh{}
print("abort:", configuration.abort)
```

### Analyzing Output
There are several options to analyze the output of a command.

#### lines()
An iterator is accessible by calling the method `lines()` on the command.

```lua
for line in sh.cat("/etc/hosts"):lines() do
  print(line)
end
```

#### stdout([filename]), stderr([filename]), combinedOutput([filename])
`stdout()`, `stderr()`, and `combinedOutput()` all returns the output of the command as a string. An optional `filename` can be given to the method, in that case the output is also written to the file. The file will be truncated.

```lua
-- print output of command
output = sh.echo("hello world"):combinedOutput("/tmp/output")
print(output)
```

In the example above will print `hello world` and it will write it to `/tmp/output`

### Glob Expansion
There is no glob expansion done on arguments, however there is a glob functionality in sh.

```lua
sh.ls(sh.glob("*.go"))
```

## `re`

```lua
local re = require "re"
```

Regular Expressions

### API

re.find , re.gsub, re.match, re.gmatch are available. These functions have the same API as Lua pattern match.
gluare uses the Go regexp package, so you can use regular expressions that are supported in the Go regexp package.

In addition, the following functions are defined:
```
gluare.quote(s string) -> string
Arguments:

s string:	a string value to escape meta characters

Returns:

string:	escaped string
gluare.quote returns a string that quotes all regular expression metacharacters inside the given text.
```

## `simplebox`

```lua
local simplebox = require "simplebox"
```

Simple encryption

### API

#### Create a new instance of simplebox with a newly generated key

```lua
local simplebox = require "simplebox"
local key = simplebox.genkey()
print("key is: " .. key)
local sb = simplebox.new()


```
