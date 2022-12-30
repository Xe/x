self:
{ config, lib, pkgs, ... }:
with lib;
let
  system = pkgs.system;
  cfg = config.xeserv.services.aegis;
  selfpkgs = self.packages.${system};
in {
  options.xeserv.services.aegis = {
    enable = mkEnableOption "Activates Aegis (unix socket prometheus proxy)";

    hostport = mkOption {
      type = types.str;
      default = "[::1]:31337";
      description = "The host:port that aegis should listen for traffic on";
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
}
