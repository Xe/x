yeet.setenv("GOOS", "linux");
yeet.setenv("GOARCH", "amd64");
yeet.setenv("CGO_ENABLED", "0");

go.build();
const fname = slug.build("todayinmarch2020");

const url = slug.push(fname);
const hash = nix.hashURL(url);

const expr = nix.expr`{ stdenv }:

stdenv.mkDerivation {
  name = "todayinmarch2020";
  src = builtins.fetchurl {
    url = ${url};
    sha256 = ${hash};
  };

  phases = "installPhase";

  installPhase = ''
    tar xf $src
    mkdir -p $out/bin
    cp bin/main $out/bin/todayinmarch2020
  '';
}
`;

file.write("/home/cadey/code/nixos-configs/pkgs/x/todayinmarch2020.nix", expr);
