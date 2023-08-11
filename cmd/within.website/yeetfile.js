yeet.setenv("GOOS", "linux");
yeet.setenv("GOARCH", "amd64");

go.build();
slug.build("within.website", {
    "config.ts": "config.ts"
});

const url = slug.push("within.website");
const hash = nix.hashURL(url);

const expr = nix.expr`{ stdenv }:

stdenv.mkDerivation {
  name = "within.website";
  src = builtins.fetchurl {
    url = ${url};
    sha256 = ${hash};
  };

  phases = "installPhase";

  installPhase = ''
    tar xf $src
    mkdir -p $out/bin
    cp web $out/bin/withinwebsite
    cp config.ts $out/config.ts
  '';
}
`;

file.write("/home/cadey/code/nixos-configs/pkgs/x/withinwebsite.nix", expr);
