["amd64", "arm64", "386"].forEach(goarch => {
  [deb, rpm, tarball].forEach(method => {
    method.build({
      name: "ingressd",
      description: "ingress for my homelab",
      homepage: "https://within.website",
      license: "CC0",
      goarch,

      documentation: {
        "LICENSE": "LICENSE",
      },

      build: ({ bin, systemd }) => {
        $`go build -o ${bin}/ingressd -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"' ./cmd/ingressd`
        file.install("./cmd/ingressd/ingressd.service", `${systemd}/ingressd.service`);
      },
    });

    method.build({
      name: "license",
      description: "software license generator",
      homepage: "https://within.website",
      license: "CC0",
      goarch,

      documentation: {
        "LICENSE": "LICENSE",
      },

      build: ({ bin, systemd }) => {
        $`go build -o ${bin}/license -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"' ./cmd/license`
      },
    });

    method.build({
      name: "quickserv",
      description: "Like python3 -m http.server but a single binary",
      homepage: "https://within.website",
      license: "CC0",
      goarch,

      documentation: {
        "LICENSE": "LICENSE",
      },

      build: ({ bin }) => {
        $`go build -o ${bin}/quickserv -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"' ./cmd/quickserv`
      },
    });

    method.build({
      name: "yeet",
      description: "Yeet out actions with maximum haste!",
      homepage: "https://within.website",
      license: "CC0",
      goarch,

      documentation: {
        "./cmd/yeet/README.md": "README.md",
        "LICENSE": "LICENSE",
      },

      build: ({ bin }) => {
        $`CGO_ENABLED=0 go build -o ${bin}/yeet -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"' ./cmd/yeet`
      },
    });

    method.build({
      name: "x",
      description: "the universal x command",
      homepage: "https://within.website",
      license: "CC0",
      goarch,

      documentation: {
        "LICENSE": "LICENSE",
      },

      build: ({ bin, systemd }) => {
        $`go build -o ${bin}/x -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"' ./cmd/x`
      },
    });
  });
});