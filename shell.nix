let
  pkgs = import <nixpkgs> {};
in
pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    golint
    golangci-lint
    goimports
    vgo2nix

    python310

    niv
  ];
}
