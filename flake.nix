{
  description = "/x/perimental code";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";

    rust-overlay = {
      url = "github:oxalica/rust-overlay";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "utils";
    };

    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.utils.follows = "utils";
    };
  };

  outputs = { self, nixpkgs, utils, gomod2nix, rust-overlay }@attrs:
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
              go = prev.go_1_19;
              buildGoModule = prev.buildGo118Module;
            })
            gomod2nix.overlays.default
            rust-overlay.overlays.default
            #(final: prev: self.packages.${system})
          ];
        };

        everything = pkgs.buildGoApplication {
          pname = "xe-x-composite";
          version = "1.0.0";
          src = ./.;
          modules = ./gomod2nix.toml;

          buildInputs = with pkgs; [
            pkg-config
            libaom
            libavif
            sqlite-interactive
          ];
        };

        copyFile = { pname, path ? pname }:
          pkgs.stdenv.mkDerivation {
            inherit pname;
            inherit (everything) version;
            src = everything;

            installPhase = ''
              mkdir -p $out/bin
              cp $src/bin/$pname $out/bin/$path
            '';
          };
      in {
        packages = rec {
          default = everything;

          license = copyFile {
            pname = "license";
            path = "xlicense";
          };
          makeMastodonApp = copyFile {
            pname = "mkapp";
            path = "make-mastodon-app";
          };
          aegis = copyFile { pname = "aegis"; };
          cadeybot = copyFile { pname = "cadeybot"; };
          hlang = copyFile { pname = "hlang"; };
          johaus = copyFile { pname = "johaus"; };
          mainsanow = copyFile { pname = "mainsanow"; };
          prefix = copyFile { pname = "prefix"; };
          quickserv = copyFile { pname = "quickserv"; };
          todayinmarch2020 = copyFile { pname = "todayinmarch2020"; };
          uploud = copyFile { pname = "uploud"; };
          within-website = copyFile { pname = "within.website"; };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            gomod2nix.packages.${system}.default
            python

            pkg-config
            libaom
            libavif
            sqlite-interactive

            cargo
            cargo-watch
            rustfmt
            rust-analyzer
            wasmtime
            binaryen
            wabt
            bloaty
            (rust-bin.stable.latest.default.override {
              extensions = [ "rust-src" ];
              targets = [ "wasm32-wasi" ];
            })
          ];
        };
      }) // {
        nixosModules = {
          overlay = { ... }: {
            nixpkgs.overlays = [
              (final: prev:
                let
                  system = prev.system;
                  selfpkgs = self.packages.${system};
                in { xeserv = selfpkgs; })
            ];
          };

          default = { ... }: {
            imports = with self.nixosModules; [
              overlay
              aegis
              todayinmarch2020
              within-website
            ];
          };

          aegis = import ./nix/aegis.nix self;
          todayinmarch2020 = import ./nix/todayinmarch2020.nix self;
          within-website = import ./nix/within-website.nix self;
        };
      };
}
