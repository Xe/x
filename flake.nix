{
  description = "/x/perimental code";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
    gomod2nix.url = "github:tweag/gomod2nix";
  };

  outputs = { self, nixpkgs, utils, gomod2nix }:
    utils.lib.eachSystem [
      "x86_64-linux"
      "aarch64-linux"
      "x86_64-darwin"
      "aarch64-darwin"
    ] (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            (final: prev: {
              go = prev.go_1_18;
              buildGoModule = prev.buildGo118Module;
            })
            gomod2nix.overlay
          ];
        };
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [ go gopls gotools go-tools gomod2nix.defaultPackage.${system} ];
        };
      });
}
