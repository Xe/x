["amd64", "arm64"].forEach((goarch) =>
  rpm.build({
    name: "ingressd",
    description: "ingress for my homelab",
    homepage: "https://within.website",
    license: "CC0",
    goarch,

    build: ({ bin, systemd }) => {
      go.build("-o", `${bin}/ingressd`);
      file.install("ingressd.service", `${systemd}/ingressd.service`);
    },
  }),
);
