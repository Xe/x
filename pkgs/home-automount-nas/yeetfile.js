rpm.build({
  name: "home-automount-nas",
  description: "Automount the NAS drive on boot",
  homepage: "https://xeiaso.net",
  license: "CC0",
  goarch: "all",

  build: ({ systemd }) => {
    file.install("./mnt-itsuki.automount", `${systemd}/mnt-itsuki.automount`);
    file.install("./mnt-itsuki.mount", `${systemd}/mnt-itsuki.mount`);
  },
});
