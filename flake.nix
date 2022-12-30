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
          default = { ... }: {
            imports = with self.nixosModules; [ aegis todayinmarch2020 within-website ];
          };

          aegis = { config, lib, pkgs, ... }:
            with lib;
            let
              system = pkgs.system;
              cfg = config.xeserv.services.aegis;
              selfpkgs = self.packages.${system};
            in {
              options.xeserv.services.aegis = {
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

          todayinmarch2020 = { config, pkgs, lib, ... }:
            with lib;
            let
              system = pkgs.system;
              selfpkgs = self.packages.${system};
              cfg = config.xeserv.services.todayinmarch2020;
            in {
              options.xeserv.services.todayinmarch2020 = {
                enable =
                  mkEnableOption "Lets you find out what day in 2020 today is";
                useACME = mkEnableOption "Enables ACME for cert stuff";

                domain = mkOption {
                  type = types.str;
                  default = "todayinmarch2020.xn--sz8hf6d.ws";
                  example = "todayinmarch2020.xn--sz8hf6d.ws";
                  description =
                    "The domain name that nginx should check against for HTTP hostnames";
                };

                sockPath = mkOption rec {
                  type = types.str;
                  default = "/srv/within/run/todayinmarch2020.sock";
                  example = default;
                  description =
                    "The unix domain socket that printerfacts should listen on";
                };
              };

              config = mkIf cfg.enable {
                users.users.todayinmarch2020 = {
                  createHome = true;
                  description = "tulpa.dev/cadey/todayinmarch2020";
                  isSystemUser = true;
                  group = "within";
                  home = "/srv/within/todayinmarch2020";
                };

                systemd.services.todayinmarch2020 = {
                  wantedBy = [ "multi-user.target" ];
                  after = [ "within-homedir.service" ];

                  serviceConfig = {
                    User = "todayinmarch2020";
                    Group = "within";
                    Restart = "on-failure";
                    WorkingDirectory = "/srv/within/todayinmarch2020";
                    RestartSec = "30s";
                    UMask = "007";
                  };

                  script = ''
                    exec ${selfpkgs.todayinmarch2020}/bin/todayinmarch2020 -socket=${cfg.sockPath}
                  '';
                };

                services.nginx.virtualHosts."todayinmarch2020" = {
                  serverName = "${cfg.domain}";
                  locations."/".proxyPass = "http://unix:${cfg.sockPath}";
                  enableACME = true;
                  extraConfig = ''
                    access_log /var/log/nginx/todayinmarch2020.access.log;
                  '';
                };
              };
            };

          within-website = { config, lib, pkgs, ... }:
            with lib;
            let
              system = pkgs.system;
              cfg = config.xeserv.services.within-website;
              selfpkgs = self.packages.${system};
            in {
              options.xeserv.services.within-website = {
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
      };
}
