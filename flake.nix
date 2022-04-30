{
  description = "/x/perimental code";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
    portable-svc.url = "git+https://tulpa.dev/cadey/portable-svc.git?ref=main";
    ckiee.url = "github:ckiee/nixpkgs?ref=gpt2simple-py-init";

    gomod2nix = {
      url = "github:tweag/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.utils.follows = "utils";
      };
  };

  outputs = { self, nixpkgs, utils, gomod2nix, portable-svc, ckiee }@attrs:
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
            (final: prev: self.packages.${system})
          ];
        };
        ckieepkgs = import ckiee { inherit system; };

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

        python =
          (ckieepkgs.python310.withPackages (ps: with ps; [ gpt-2-simple ]));
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
          robocadey-gpt2 = pkgs.writeShellScriptBin "robocadey-gpt2" ''
            ${python}/bin/python3 ${./mastodon/robocadey/gpt2/main.py}
          '';
          robocadey-psvc = let
            service = pkgs.substituteAll {
              name = "robocadey.service";
              src = ./run/robocadey.service.in;
              robocadey = self.packages.${system}.robocadey;
            };
            gpt2-service = pkgs.substituteAll {
              name = "robocadey-gpt2.service";
              src = ./run/robocadey-gpt2.service.in;
              inherit python;
              main = ./mastodon/robocadey/gpt2/main.py;
            };
          in pkgs.portableService {
            inherit (self.packages.${system}.robocadey) version;
            name = "robocadey";
            description = "Robotic twitter shitposting bot";
            units = [ service gpt2-service ./run/robocadey-gpt2.socket ];
            symlinks = [{
              object = "${pkgs.cacert}/etc/ssl";
              symlink = "/etc/ssl";
            }];
          };
        };

        nixosModules.robocadey = { config, lib, pkgs, ... }:
          with lib;
          let
            system = pkgs.system;
            cfg = config.xeserv.services.robocadey;
            selfpkgs = self.packages.${system};
          in {
            options.xeserv.services.robocadey = {
              enable = mkEnableOption "Activates the printerfacts server";

              pathToModel = mkOption {
                type = types.str;
                default = "/srv/models/robocadey_gpt2.raw";
                description = "model squashfs volume location";
              };
            };

            config = mkIf cfg.enable {
              systemd.mounts = [{
                type = "squashfs";
                what = cfg.pathToModel;
                where = "/var/lib/private/xeserv.robocadey-gpt2/checkpoint";
                options = "ro,relatime,errors=continue";
              }];
              systemd.services = {
                "robocadey" = {
                  wantedBy = [ "multi-user.target" ];
                  description = "RoboCadey";

                  serviceConfig = {
                    DynamicUser = "true";
                    ExecStart = "${selfpkgs.robocadey}/bin/robocadey";
                    WorkingDirectory = "/var/lib/private/xeserv.robocadey";
                    StateDirectory = "xeserv.robocadey";
                    CacheDirectory = "xeserv.robocadey";
                  };
                };
                "robocadey-gpt2" = {
                  wantedBy = [ "multi-user.target" ];
                  description = "RoboCadey GPT2 sidecar";

                  serviceConfig = {
                    DynamicUser = "true";
                    ExecStart = "${selfpkgs.robocadey-gpt2}/bin/robocadey-gpt2";
                    WorkingDirectory = "/var/lib/private/xeserv.robocadey-gpt2";
                    StateDirectory = "xeserv.robocadey-gpt2";
                    CacheDirectory = "xeserv.robocadey-gpt2";
                  };
                };
              };
              systemd.sockets."robocadey-gpt2" = {
                description = "RoboCadey GPT-2 activation socket";
                partOf = "robocadey-gpt2.service";
                listenStreams = [ "/run/robocadey-gpt2.sock" ];
              };
            };
          };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            gomod2nix.defaultPackage.${system}
            python
          ];
        };
      });
}
