let
  pkgs = import <nixpkgs> {};
in
pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    golint
    gometalinter
    goimports
    vgo2nix

    niv
  ];
}
