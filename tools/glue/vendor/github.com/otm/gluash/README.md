# gluash
glush is a interface to call any program as it were a function. Programs are executed asynchronously to enable streaming of data in pipes.

# Installation
```
go get github.com/otm/gluash
```

# Usage
``` lua
import "github.com/yuin/gopher-lua"
import "github.com/otm/gluash"

func main() {
    L := lua.NewState()
    defer L.Close()

    L.PreloadModule("sh", gluash.Loader)

    if err := L.DoString(`

      local sh = require("sh")

      sh.echo("hello", "world"):print()

    `); err != nil {
        panic(err)
    }
}
```

# API
In all discussions bellow the imported module will be referred to as `sh`.

Commands are called just like functions, executed on the sh module.

```lua
sh.ls("/")
```

For commands that have exotic names, names that are reserved words, or to execute absolute or relative paths call the sh module directly.

```lua
sh("/bin/ls", "/")
```

### Multiple Arguments
Commands with multiple arguments have to be invoked with a separate string for each argument.

```lua
-- this works
sh.ls("-la", "/")

-- this does not work
sh.ls("-la /")
```

### Piping
Piping in sh is done almost like piping in the shell. Just call next command as a method on the previous command.

``` lua
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

``` lua
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
