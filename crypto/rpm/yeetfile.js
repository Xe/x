rpm.build({
  name: "xeiaso.net-keys",
  description: "Public keys for xeiaso.net RPM packages",
  homepage: "https://xeiaso.net",
  license: "CC0",
  goarch: "all",

  build: (out) => {
    yeet.run(`mkdir`, `-p`, `${out}/etc/pki/rpm-gpg`);
    yeet.run(
      `cp`,
      `xeiaso.net-keys.asc`,
      `${out}/etc/pki/rpm-gpg/xeiaso.net-keys`,
    );
  },
});
