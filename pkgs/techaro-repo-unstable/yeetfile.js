const name = "techaro-repo-unstable";
const description = "pkgs.techaro.lol unstable packages";
const homepage = "https://techaro.lol";
const license = "CC0";
const goarch = "all";
const version = "1.0.1";

deb.build({
  name,
  description,
  homepage,
  license,
  goarch,
  version,

  build: (out) => {
    file.install(
      "./techaro-pkgs-unstable.pub.asc",
      `${out}/etc/apt/keyrings/techaro-pkgs-unstable.pub.asc`,
    );
    file.install(
      "./techaro-unstable.list",
      `${out}/etc/apt/sources.list.d/techaro-unstable.list`,
    );
  },
});

rpm.build({
  name,
  description,
  homepage,
  license,
  goarch,
  version,

  build: (out) => {
    file.install(
      "./techaro-pkgs-unstable.pub.asc",
      `${out}/etc/pki/rpm-gpg/techaro-pkgs-unstable.asc`,
    );
    file.install(
      "./techaro-unstable.repo",
      `${out}/etc/yum.repos.d/techaro-unstable.repo`,
    );
  },
});
