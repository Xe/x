self:
{ config, lib, pkgs, ... }:
with lib;
let
  system = pkgs.system;
  cfg = config.xeserv.services.within-website;
  selfpkgs = self.packages.${system};
in {
  options.xeserv.services.within-website = {
    enable = mkEnableOption "Enables the within.website import redirector";

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
      description = "the package containing the within.website binary";
    };
  };
  config = mkIf cfg.enable {
    systemd.services.within-website = {
      wantedBy = [ "multi-user.target" ];
      serviceConfig = {
        DynamicUser = "true";
        Restart = "always";
        RestartSec = "30s";
        ExecStart =
          "${cfg.package}/bin/within.website --port=${toString cfg.port} --tyson-config ${../cmd/within.website/config.ts}";
      };
    };

    services.nginx.virtualHosts."withinwebsite" = {
      serverName = "${cfg.domain}";
      locations."/".proxyPass = "http://127.0.0.1:${toString cfg.port}";
      forceSSL = true;
      useACMEHost = "${cfg.domain}";
      extraConfig = ''
        access_log /var/log/nginx/withinwebsite.access.log;
      '';
    };
  };
}
