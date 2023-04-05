self:
{ config, lib, pkgs, ... }:
with lib;
let
  system = pkgs.system;
  cfg = config.xeserv.services.vest-pit-near;
  selfpkgs = self.packages.${system};
in {
  options.xeserv.services.vest-pit-near = {
    enable = mkEnableOption "Enables the vest-pit-near Mullvad checker";

    domain = mkOption {
      type = types.str;
      default = "vest-pit-near";
      example = "vest-pit-near";
      description =
        "The domain name that nginx should check against for HTTP hostnames";
    };

    package = mkOption {
      type = types.package;
      default = selfpkgs.vest-pit-near;
      description = "the package containing the vest-pit-near binary";
    };
  };
  config = mkIf cfg.enable {
    systemd.services.vest-pit-near = {
      wantedBy = [ "multi-user.target" ];
      after = [ "within-homedir.service" ];
      environment.STATE_DIR = "/var/lib/private/vest-pit-near";
      serviceConfig = {
        DynamicUser = "true";
        SupplementaryGroups = [ "docker" ];
        Restart = "always";
        RestartSec = "30s";
        ExecStart =
          "${cfg.package}/bin/vest-pit-near";
        RuntimeDirectory = "vest-pit-near";
        RuntimeDirectoryMode = "0755";
        StateDirectory = "vest-pit-near";
        StateDirectoryMode = "0700";
        CacheDirectory = "vest-pit-near";
        CacheDirectoryMode = "0750";
      };
    };
  };
}
