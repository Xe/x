rpm.build({
  name: "relayd",
  description: "TLS termination and client fingerprinting",
  homepage: "https://within.website",
  license: "CC0",
  goarch: "amd64",

  documentation: {
    LICENSE: "LICENSE",
  },

  configFiles: {
    "cmd/relayd/relayd.env": "/etc/within.website/x/relayd.env",
  },

  build: ({ bin, systemd }) => {
    $`go build -o ${bin}/relayd -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"' ./cmd/relayd`;
    file.install("./cmd/relayd/relayd.service", `${systemd}/relayd.service`);
  },
});
