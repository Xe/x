yeet.setenv("GOOS", "linux");
yeet.setenv("GOARCH", "amd64");
yeet.setenv("CGO_ENABLED", "0");

go.build();
const fname = slug.build("johaus");

const url = slug.push(fname);
const hash = nix.hashURL(url);

const expr = nix.expr`{ stdenv }:

stdenv.mkDerivation {
  name = "johaus";
  src = builtins.fetchurl {
    url = ${url};
    sha256 = ${hash};
  };

  phases = "installPhase";

  installPhase = ''
    tar xf $src
    mkdir -p $out/bin
    cp bin/main $out/bin/johaus
  '';
}
`;

file.write("/home/cadey/code/nixos-configs/pkgs/x/johaus.nix", expr);
