go.install();

yeet.setenv("GOARM", "7");

[
    // "386",
    "amd64",
    // "arm",
    "arm64",
    // "loong64",
    // "mips",
    // "mips64",
    // "mips64le",
    // "mipsle",
    // "ppc64",
    // "ppc64le",
    // "riscv64",
    // "s390x",
].forEach(goarch => {
    [deb, rpm, tarball].forEach(method => method.build({
        name: "yeet",
        description: "Yeet out actions with maximum haste!",
        homepage: "https://within.website",
        license: "CC0",
        goarch,

        documentation: {
            "README.md": "README.md",
            "../../LICENSE": "LICENSE",
        },

        build: ({ bin }) => {
            $`CGO_ENABLED=0 go build -o ${bin}/yeet -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"'`
        },
    }))
})