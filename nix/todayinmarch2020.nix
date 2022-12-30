self:
{ config, pkgs, lib, ... }:
with lib;
let
  system = pkgs.system;
  selfpkgs = self.packages.${system};
  cfg = config.xeserv.services.todayinmarch2020;
in {
  options.xeserv.services.todayinmarch2020 = {
    enable = mkEnableOption "Lets you find out what day in 2020 today is";
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
      description = "The unix domain socket that printerfacts should listen on";
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
}
