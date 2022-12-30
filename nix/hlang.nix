self:
{ config, pkgs, lib, ... }:

with lib;

let cfg = config.xeserv.services.hlang;
in {
  options.xeserv.services.hlang = {
    enable = mkEnableOption "Enables the H programming language";
    useACME = mkEnableOption "Enables ACME for cert stuff";

    domain = mkOption {
      type = types.str;
      default = "hlang.akua";
      example = "h.christine.website";
      description =
        "The domain name that nginx should check against for HTTP hostnames";
    };

    port = mkOption {
      type = types.int;
      default = 38288;
      example = 9001;
      description = "The port number hlang should listen on for HTTP traffic";
    };

    sockPath = mkOption rec {
      type = types.str;
      default = "/srv/within/run/hlang.sock";
      example = default;
      description = "The unix domain socket that printerfacts should listen on";
    };
  };

  config = mkIf cfg.enable {
    users.users.hlang = {
      createHome = true;
      description = "tulpa.dev/cadey/hlang";
      isSystemUser = true;
      group = "within";
      home = "/srv/within/hlang";
    };

    systemd.services.hlang = {
      wantedBy = [ "multi-user.target" ];
      after = [ "within-homedir.service" ];

      serviceConfig = {
        User = "hlang";
        Group = "within";
        Restart = "on-failure";
        WorkingDirectory = "/srv/within/hlang";
        RestartSec = "30s";
        UMask = "007";
      };

      script = let h = self.packages.${pkgs.system}.hlang;
      in ''
        export PATH=${pkgs.wabt}/bin:$PATH
        exec ${h}/bin/hlang -port=${toString cfg.port} -sockpath=${cfg.sockPath}
      '';
    };

    services.cfdyndns = mkIf cfg.useACME { records = [ "${cfg.domain}" ]; };

    services.nginx.virtualHosts."hlang" = {
      serverName = "${cfg.domain}";
      locations."/".proxyPass = "http://unix:${cfg.sockPath}";
      forceSSL = cfg.useACME;
      useACMEHost = "christine.website";
      extraConfig = ''
        access_log /var/log/nginx/hlang.access.log;
      '';
    };
  };
}
