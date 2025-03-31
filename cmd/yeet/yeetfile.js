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
    "riscv64",
    // "ppc64",
    // "ppc64le",
    // "s390x",
]
    .forEach(goarch => {
        [
            deb,
            rpm,
        ]
            .forEach(method => method.build({
                name: "yeet",
                description: "Yeet out actions with maximum haste!",
                homepage: "https://within.website",
                license: "CC0",
                goarch,

                documentation: {
                    "README.md": "README.md",
                    "../../LICENSE": "LICENSE",
                },

                build: (out) => {
                    go.build("-o", `${out}/usr/bin/yeet`);
                },
            }));
    });