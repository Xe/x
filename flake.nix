{
  description = "/x/perimental code";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
    gomod2nix.url = "github:tweag/gomod2nix";
    portable-svc.url = "git+https://tulpa.dev/cadey/portable-svc.git?ref=main";
  };

  outputs = { self, nixpkgs, utils, gomod2nix, portable-svc }:
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
            portable-svc.overlay
          ];
        };

        everything = pkgs.buildGoApplication {
          pname = "xe-x-composite";
          version = "1.0.0";
          src = ./.;
          modules = ./gomod2nix.toml;
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
          prefix = copyFile { pname = "prefix"; };
          quickserv = copyFile { pname = "quickserv"; };
          within-website = copyFile { pname = "within.website"; };
          johaus = copyFile { pname = "johaus"; };
          cadeybot = copyFile { pname = "cadeybot"; };
          mainsanow = copyFile { pname = "mainsanow"; };
          importer = copyFile {
            pname = "importer";
            path = "cadeybot-importer";
          };

          robocadey = copyFile { pname = "robocadey"; };
          robocadey-psvc = let
            preflight = pkgs.writeShellApplication {
              name = "cadeybot-preflight";
              runtimeInputs = with pkgs; [ jq coreutils gnugrep gnused ];
              text = builtins.readFile ./run/robocadey.preflight.sh;
            };
            service = pkgs.substituteAll {
              name = "robocadey.service";
              src = ./run/robocadey.service.in;
              inherit preflight;
              robocadey = self.packages.${system}.robocadey;
            };
          in pkgs.portableService {
            inherit (self.packages.${system}.robocadey) version;
            name = "robocadey";
            description = "Robotic twitter shitposting bot";
            units = [ service ];
            symlinks = [{
              object = "${pkgs.cacert}/etc/ssl";
              symlink = "/etc/ssl";
            }];
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            gomod2nix.defaultPackage.${system}
          ];
        };
      });
}
