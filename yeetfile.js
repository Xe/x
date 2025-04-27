yeet.setenv("KO_DOCKER_REPO", "ghcr.io/xe/x");
yeet.setenv("SOURCE_DATE_EPOCH", $`git log -1 --format='%ct'`.trim());
yeet.setenv("VERSION", git.tag());

programs = [
  "aerial",
  "amano",
  "aura",
  "future-sight",
  "httpdebug",
  "quickserv",
  "relayd",
  "reverseproxyd",
  "sapientwindex",
  "stickers",
  "todayinmarch2020",
  "uncle-ted",
  "within.website",
].join(",");

$`ko build --platform=all --base-import-paths --tags=latest,${git.tag()} ./cmd/{${programs}}`;

yeet.setenv("CGO_ENABLED", "0");

["amd64", "arm64"].forEach((goarch) => {
  [deb, rpm, tarball].forEach((method) => {
    method.build({
      name: "ingressd",
      description: "ingress for my homelab",
      homepage: "https://within.website",
      license: "CC0",
      goarch,

      documentation: {
        LICENSE: "LICENSE",
      },

      build: ({ bin, systemd }) => {
        $`go build -o ${bin}/ingressd -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"' ./cmd/ingressd`;
        file.install(
          "./cmd/ingressd/ingressd.service",
          `${systemd}/ingressd.service`,
        );
      },
    });

    method.build({
      name: "license",
      description: "software license generator",
      homepage: "https://within.website",
      license: "CC0",
      goarch,

      documentation: {
        LICENSE: "LICENSE",
      },

      build: ({ bin }) => {
        $`go build -o ${bin}/license -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"' ./cmd/license`;
      },
    });

    method.build({
      name: "quickserv",
      description: "Like python3 -m http.server but a single binary",
      homepage: "https://within.website",
      license: "CC0",
      goarch,

      documentation: {
        LICENSE: "LICENSE",
      },

      build: ({ bin }) => {
        $`go build -o ${bin}/quickserv -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"' ./cmd/quickserv`;
      },
    });

    method.build({
      name: "relayd",
      description: "TLS termination and client fingerprinting",
      homepage: "https://within.website",
      license: "CC0",
      goarch,

      documentation: {
        LICENSE: "LICENSE",
      },

      configFiles: {
        "cmd/relayd/relayd.env": "/etc/within.website/x/relayd.env",
      },

      build: ({ bin, systemd }) => {
        $`go build -o ${bin}/relayd -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"' ./cmd/relayd`;
        file.install(
          "./cmd/relayd/relayd.service",
          `${systemd}/relayd.service`,
        );
      },
    });

    method.build({
      name: "uploud",
      description: "Upload images to the cloud!",
      homepage: "https://within.website",
      license: "CC0",
      goarch,

      documentation: {
        LICENSE: "LICENSE",
      },

      build: ({ bin }) => {
        $`go build -o ${bin}/uploud -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"' ./cmd/uploud`;
      },
    });

    method.build({
      name: "x",
      description: "the universal x command",
      homepage: "https://within.website",
      license: "CC0",
      goarch,

      documentation: {
        LICENSE: "LICENSE",
      },

      build: ({ bin, systemd }) => {
        $`go build -o ${bin}/x -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"' ./cmd/x`;
      },
    });
  });
});
