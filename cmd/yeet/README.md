# yeet

Yeet out actions with maximum haste! Declare your build instructions as small JavaScript snippets and let er rip!

## Usage

To install the current program with `go install`:

```js
// yeetfile.js
go.install();
```

## Available functions

Yeet uses [goja](https://pkg.go.dev/github.com/dop251/goja#section-readme) to execute JavaScript. As such, it does not have access to NPM or other external JavaScript libraries. You also cannot import code/data from other files. These are not planned for inclusion into yeet. If functionality is required, it should be added to yeet itself.

To make it useful, yeet exposes a bunch of helper objects full of tools. These tools fall in a few categories, each has its own section.

### `docker`

Aliases for `docker` commands.

#### `docker.build`

An alias for the `docker build` command. Builds a docker image in the current working directory's Dockerfile.

Usage:

`docker.build(tag);`

```js
docker.build("ghcr.io/xe/site/bin");
docker.push("ghcr.io/xe/site/bin");
```

#### `docker.load`

Loads an exported docker image by path into the local docker daemon. This is most useful when combined with tools like `nix.build`.

Usage:

`docker.load(path)`

```js
nix.build(".#docker.xedn");
docker.load("./result");
docker.push("registry.fly.io/xedn:latest");
fly.deploy();
```

#### `docker.push`

Pushes a docker image to a registry. Analogous to `docker push` in the CLI.

Usage:

`docker.push(tag);`

```js
docker.build("ghcr.io/xe/site/bin");
docker.push("ghcr.io/xe/site/bin");
```

### `file`

Helpers for filesystem access.

#### `file.copy`

Copies a file from one place to another. Analogous to the `cp` command on Linux. Automatically creates directories in the `dest` path if they don't exist.

Usage:

`file.copy(src, dest);`

```js
file.copy("LICENSE", `${out}/usr/share/doc/LICENSE`);
```

#### `file.read`

Reads a file into memory and returns it as a string.

Usage:

`file.read(path);`

```js
const version = file.read("VERSION");
```

#### `file.write`

Writes the contents of the `data` string to a file with mode `0660`.

Usage:

`file.write(path, data);`

```js
file.write("VERSION", git.tag());
```

### `fly`

Automation for [flyctl](https://github.com/superfly/flyctl). Soon this will also let you manage Machines with the [Machines API](https://docs.machines.dev).

#### `fly.deploy`

Runs the `fly deploy` command for you.

Usage:

`fly.deploy();`

```js
docker.build("registry.fly.io/foobar");
docker.push("registry.fly.io/foobar");
fly.deploy();
```

### `git`

Helpers for the Git version control system.

#### `git.repoRoot`

Returns the repository root as a string.

`git.repoRoot();`

```js
const repoRoot = git.repoRoot();

file.copy(`${repoRoot}/LICENSE`, `${out}/usr/share/doc/yeet/LICENSE`);
```

#### `git.tag`

Returns the output of `git describe --tags`. Useful for getting the "current version" of the repo, where the current version will likely be different forward in time than it is backwards in time.

Usage:

`git.tag();`

```js
const version = git.tag();
```

### `go`

Helpers for the Go programming language.

#### `go.build`

Runs `go build` in the current working directory with any extra arguments passed in. This is useful for building and installing Go programs in an RPM build context.

Usage:

`go.build(args);`

```js
go.build("-o", `${out}/usr/bin/`);
```

#### `go.install`

Runs `go install`. Not useful for cross-compilation.

Usage:

`go.install();`

```js
go.install();
```

### `nix`

Automation for running Nix ecosystem tooling.

#### `nix.build`

Runs `nix build` against a given flakeref.

Usage:

`nix.build(flakeref);`

```js
nix.build(".#docker");
docker.load("./result");
```

#### `nix.eval`

A tagged template that helps you build Nix expressions safely from JavaScript and then evaluates them. See my [nixexpr blogpost](https://xeiaso.net/blog/nixexpr/) for more information about how this works.

Usage:

```js
const glibcPath = nix.eval`let pkgs = import <nixpkgs>; in pkgs.glibc`;
```

#### `nix.expr`

A tagged template that helps you build Nix expressions safely from JavaScript. See my [nixexpr blogpost](https://xeiaso.net/blog/nixexpr/) for more information about how this works.

Usage:

```js
go.build();
const fname = slug.build("todayinmarch2020");

const url = slug.push(fname);
const hash = nix.hashURL(url);

const expr = nix.expr`{ stdenv }:

stdenv.mkDerivation {
  name = "todayinmarch2020";
  src = builtins.fetchurl {
    url = ${url};
    sha256 = ${hash};
  };

  phases = "installPhase";

  installPhase = ''
    tar xf $src
    mkdir -p $out/bin
    cp bin/main $out/bin/todayinmarch2020
  '';
}
`;

file.write(`${repoRoot}/pkgs/x/todayinmarch2020.nix`, expr);
```

#### `nix.hashURL`

Hashes the contents of a given URL and returns the `sha256` SRI form. Useful when composing Nix expressions with the `nix.expr` tagged template.

Usage:

`nix.hashURL(url);`

```js
const hash = nix.hashURL("https://whatever.com/some_file.tgz");
```

### `rpm`

Helpers for building RPM packages and docker images out of a constellation of RPM packages.

#### `rpm.build`

Builds an RPM package with a descriptor object. See the RPM packages section for more information. The important part of this is your `build` function. The `build` function is what will turn your package source code into an executable in `out` somehow. Everything in `out` corresponds 1:1 with paths in the resulting RPM.

The resulting RPM path will be returned as a string.

Usage:

`rpm.build(package);`

```js
["amd64", "arm64"].forEach((goarch) =>
  rpm.build({
    name: "yeet",
    description: "Yeet out actions with maximum haste!",
    homepage: "https://within.website",
    license: "CC0",
    goarch,

    build: (out) => {
      go.build("-o", `${out}/usr/bin/`);
    },
  })
);
```

### `yeet`

This contains various "other" functions that don't have a good place to put them.

#### `yeet.cwd`

The current working directory. This is a constant value and is not updated at runtime.

Usage:

```js
log.println(yeet.cwd);
```

#### `yeet.dateTag`

A constant string representing the time that yeet was started in UTC. It is formatted in terms of `YYYYmmDDhhMM`. This is not updated at runtime. You can use it for a "unique" value per invocation of yeet (assuming you aren't a time traveler).

Usage:

```js
docker.build(`ghcr.io/xe/site/bin:${git.tag()}-${yeet.dateTag}`);
```

#### `yeet.run` / `yeet.runcmd`

Runs an arbitrary command and returns any output as a string.

Usage:

`yeet.run(cmd, arg1, arg2, ...);`

```js
yeet.run(
  "protoc",
  "--proto-path=.",
  `--proto-path=${git.repoRoot()}/proto`,
  "foo.proto"
);
```

#### `yeet.setenv`

Sets an environment variable for the process yeet is running in and all children.

Usage:

`yeet.setenv(key, val);`

```js
yeet.setenv("GOOS", "linux");
```

#### `yeet.goos` / `yeet.goarch`

The GOOS/GOARCH value that yeet was built for. This typically corresponds with the OS and CPU architecture that yeet is running on.

## Building RPM Packages

When using the `rpm.build` function, you can create RPM packages from arbitrary yeet expressions. This allows you to create RPM packages from a macOS or other Linux system. As an example, here is how the yeet RPMs are built:

```js
["amd64", "arm64"].forEach((goarch) =>
  rpm.build({
    name: "yeet",
    description: "Yeet out actions with maximum haste!",
    homepage: "https://within.website",
    license: "CC0",
    goarch: goarch,

    build: (out) => {
      go.build("-o", `${out}/usr/bin/`);
    },
  })
);
```

### Build settings

The following settings are supported:

| Name          | Example                                    | Description                                                                                                                                  |
| :------------ | :----------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------- |
| `name`        | `xeiaso.net-yeet`                          | The unique name of the package.                                                                                                              |
| `version`     | `1.0.0`                                    | The version of the package, if not set then it will be inferred from the git version.                                                        |
| `description` | `Yeet out scripts with haste!`             | The human-readable description of the package.                                                                                               |
| `homepage`    | `https://xeiaso.net`                       | The URL for the homepage of the package.                                                                                                     |
| `group`       | `Network`                                  | If set, the RPM group that this package belongs to.                                                                                          |
| `license`     | `MIT`                                      | The license that the contents of this package is under.                                                                                      |
| `goarch`      | `amd64` / `arm64`                          | The GOARCH value corresponding to the architecture that the RPM is being built for. If you want to build a `noarch` package, put `any` here. |
| `replaces`    | `["foo", "bar"]`                           | Any packages that this package conflicts with or replaces.                                                                                   |
| `depends`     | `["foo", "bar"]`                           | Any packages that this package depends on (such as C libraries for CGo code).                                                                |
| `emptyDirs`   | `["/var/lib/yeet"]`                        | Any empty directories that should be created when the package is installed.                                                                  |
| `configFiles` | `{"./.env.example": "/var/lib/yeet/.env"}` | Any configuration files that should be copied over on install, but managed by administrators after installation.                             |

## Support

For support, please [subscribe to me on Patreon](https://patreon.com/cadey) and ask in the `#yeet` channel. You may open GitHub issues if you wish, but I do not often look at them.
