{
  description = "/x/perimental code";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
    ckiee.url = "github:ckiee/nixpkgs?ref=gpt2simple-py-init";

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

  outputs = { self, nixpkgs, utils, gomod2nix, ckiee, rust-overlay
    }@attrs:
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
        ckieepkgs = import ckiee { inherit system; };

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
          aegis = copyFile { pname = "aegis"; };
          prefix = copyFile { pname = "prefix"; };
          quickserv = copyFile { pname = "quickserv"; };
          within-website = copyFile { pname = "within.website"; };
          johaus = copyFile { pname = "johaus"; };
          cadeybot = copyFile { pname = "cadeybot"; };
          mainsanow = copyFile { pname = "mainsanow"; };
          uploud = copyFile { pname = "uploud"; };
          importer = copyFile {
            pname = "importer";
            path = "cadeybot-importer";
          };
        };

        nixosModules = {
          aegis = { config, lib, pkgs, ... }:
            with lib;
            let
              system = pkgs.system;
              cfg = config.xeserv.services.robocadey;
              selfpkgs = self.packages.${system};
            in {
              options.within.services.aegis = {
                enable = mkEnableOption
                  "Activates Aegis (unix socket prometheus proxy)";

                hostport = mkOption {
                  type = types.str;
                  default = "[::1]:31337";
                  description =
                    "The host:port that aegis should listen for traffic on";
                };

                sockdir = mkOption {
                  type = types.str;
                  default = "/srv/within/run";
                  example = "/srv/within/run";
                  description = "The folder that aegis will read from";
                };
              };

              config = mkIf cfg.enable {
                users.users.aegis = {
                  createHome = true;
                  description = "tulpa.dev/cadey/aegis";
                  isSystemUser = true;
                  group = "within";
                  home = "/srv/within/aegis";
                };

                systemd.services.aegis = {
                  wantedBy = [ "multi-user.target" ];

                  serviceConfig = {
                    User = "aegis";
                    Group = "within";
                    Restart = "on-failure";
                    WorkingDirectory = "/srv/within/aegis";
                    RestartSec = "30s";
                  };

                  script = let aegis = selfpkgs.aegis;
                  in ''
                    exec ${aegis}/bin/aegis -sockdir="${cfg.sockdir}" -hostport="${cfg.hostport}"
                  '';
                };
              };
            };

          robocadey = { config, lib, pkgs, ... }:
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
                    after = [ "robocadey-gpt2.socket" ];

                    serviceConfig = {
                      Restart = "always";
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
                      Restart = "always";
                      DynamicUser = "true";
                      ExecStart =
                        "${selfpkgs.robocadey-gpt2}/bin/robocadey-gpt2";
                      WorkingDirectory =
                        "/var/lib/private/xeserv.robocadey-gpt2";
                      StateDirectory = "xeserv.robocadey-gpt2";
                      CacheDirectory = "xeserv.robocadey-gpt2";
                    };
                  };
                };
                systemd.sockets."robocadey-gpt2" = {
                  description = "RoboCadey GPT-2 activation socket";
                  partOf = [ "robocadey-gpt2.service" ];
                  listenStreams = [ "/run/robocadey-gpt2.sock" ];
                };
              };
            };

          "within.website" = { config, lib, pkgs, ... }:
            with lib;
            let
              system = pkgs.system;
              cfg = config.xeserv.services.within-website;
              selfpkgs = self.packages.${system};
            in {
              options.xeserv.services.withinwebsite = {
                enable =
                  mkEnableOption "Enables the within.website import redirector";

                domain = mkOption {
                  type = types.str;
                  default = "within.website";
                  example = "within.website";
                  description =
                    "The domain name that nginx should check against for HTTP hostnames";
                };

                port = mkOption {
                  type = types.int;
                  default = 52838;
                  example = 9001;
                  description =
                    "The port number withinwebsite should listen on for HTTP traffic";
                };

                package = mkOption {
                  type = types.package;
                  default = selfpkgs.within-website;
                  description =
                    "the package containing the within.website binary";
                };
              };
              config = mkIf cfg.enable {
                systemd.services.within-website = {
                  serviceConfig = {
                    DynamicUser = "true";
                    DynamicGroup = "true";
                    Restart = "always";
                    RestartSec = "30s";
                    ExecStart = "${cfg.package}/bin/within.website --port=${
                        toString cfg.port
                      }";
                  };
                };

                services.nginx.virtualHosts."withinwebsite" = {
                  serverName = "${cfg.domain}";
                  locations."/".proxyPass =
                    "http://127.0.0.1:${toString cfg.port}";
                  forceSSL = true;
                  useACMEHost = "${cfg.domain}";
                  extraConfig = ''
                    access_log /var/log/nginx/withinwebsite.access.log;
                  '';
                };
              };
            };
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
      });
}
