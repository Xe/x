["amd64", "arm64"].forEach((goarch) => {
  [deb, rpm, tarball].forEach((method) =>
    method.build({
      name: "quickserv",
      description: "Like python3 -m http.server but a single binary",
      homepage: "https://within.website",
      license: "CC0",
      goarch,

      documentation: {
        "../../LICENSE": "LICENSE",
      },

      build: ({ bin }) => {
        $`go build -o ${bin}/quickserv -ldflags '-s -w -extldflags "-static" -X "within.website/x.Version=${git.tag()}"'`;
      },
    }),
  );
});
