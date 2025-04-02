go.install();

yeet.setenv("GOARM", "7");
yeet.setenv("CGO_ENABLED", "0");

$`CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -o ./var/yeet -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"'`;

["amd64", "arm64"].forEach(goarch => {
    [deb, rpm].forEach(method => method.build({
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