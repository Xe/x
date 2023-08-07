self:
{ config, lib, pkgs, ... }:
with lib;
let
  system = pkgs.system;
  cfg = config.xeserv.services.sanguisuga;
  selfpkgs = self.packages.${system};
in {
  options.xeserv.services.sanguisuga = {
    enable = mkEnableOption "Enables the sanguisuga torrent leeching agent";

    package = mkOption {
      type = types.package;
      default = selfpkgs.sanguisuga;
      description = "the package containing the sanguisuga binary";
    };
  };
  config = mkIf cfg.enable {
    users.users.sanguisuga = {
      isSystemUser = true;
      group = "sanguisuga";
      extraGroups = ["within"];
      createHome = true;
    };
    users.groups.sanguisuga = {};
    systemd.services.sanguisuga = {
      wantedBy = [ "multi-user.target" ];
      environment = {
        HOME = "/var/lib/sanguisuga";
        STATE_DIR = "/var/lib/sanguisuga";
      };
      path = with pkgs; [ docker ];
      serviceConfig = {
        User = "sanguisuga";
        Group = "sanguisuga";
        Restart = "always";
        RestartSec = "30s";
        ExecStart = "${cfg.package}/bin/sanguisuga";
        RuntimeDirectory = "sanguisuga";
        RuntimeDirectoryMode = "0755";
        CacheDirectory = "sanguisuga";
        CacheDirectoryMode = "0750";
        WorkingDirectory = "/var/lib/sanguisuga";
      };
    };
  };
}
