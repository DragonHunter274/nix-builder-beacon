{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.services.nix-builder-beacon;

  package = pkgs.callPackage ../. { };

  defaultHostKeyFile =
    let
      hostKeys = config.services.openssh.hostKeys or [ ];
      ed25519Keys = builtins.filter (k: k.type == "ed25519") hostKeys;
      preferred =
        if ed25519Keys != [ ] then
          builtins.head ed25519Keys
        else if hostKeys != [ ] then
          builtins.head hostKeys
        else
          null;
    in
    if preferred == null then null else "${preferred.path}.pub";

in
{
  options.services.nix-builder-beacon = {
    package = lib.mkOption {
      type = lib.types.package;
      default = package;
      defaultText = lib.literalExpression "pkgs.nix-builder-beacon";
      description = ''
        nix-builder-beacon package to use.
      '';
    };

    advert = {
      enable = lib.mkEnableOption "nix-builder-beacon advert service";

      port = lib.mkOption {
        type = lib.types.port;
        default = 22;
        description = "SSH port to advertise.";
      };

      hostname = lib.mkOption {
        type = lib.types.nullOr lib.types.str;
        default = null;
        description = "Hostname to advertise. Defaults to the local machine hostname.";
      };

      systems = lib.mkOption {
        type = lib.types.listOf lib.types.str;
        default = [ pkgs.stdenv.hostPlatform.system ];
        defaultText = lib.literalExpression "[ pkgs.stdenv.hostPlatform.system ]";
        example = [
          "x86_64-linux"
          "aarch64-linux"
        ];
        description = "Nix systems this machine can build for.";
      };

      maxJobs = lib.mkOption {
        type = lib.types.ints.positive;
        default = 1;
        description = "Maximum number of concurrent build jobs to advertise.";
      };

      speedFactor = lib.mkOption {
        type = lib.types.ints.positive;
        default = 1;
        description = "Relative speed factor to advertise.";
      };

      supportedFeatures = lib.mkOption {
        type = lib.types.listOf lib.types.str;
        default = [ ];
        example = [
          "kvm"
          "big-parallel"
        ];
        description = "Supported system features to advertise.";
      };

      mandatoryFeatures = lib.mkOption {
        type = lib.types.listOf lib.types.str;
        default = [ ];
        description = "Mandatory system features to advertise.";
      };

      sshUser = lib.mkOption {
        type = lib.types.str;
        default = "nix-ssh";
        description = ''
          SSH user to advertise for connecting to this builder. Defaults to
          `nix-ssh`, matching `nix.sshServe`'s restricted build user.
        '';
      };

      hostKeyFile = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = defaultHostKeyFile;
        defaultText = lib.literalExpression ''
          the first configured "services.openssh.hostKeys" public key file, if any
        '';
        description = ''
          Path to this machine's SSH host public key file, advertised so
          discoverers can pin it in the machines file instead of trusting the
          host on first connect.
        '';
      };
    };

    sshServe = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = cfg.advert.enable;
        defaultText = lib.literalExpression "config.services.nix-builder-beacon.advert.enable";
        description = ''
          Whether to enable `nix.sshServe` and authorize `authorizedKeys` to
          build here. Disable if you want to advertise this machine but manage
          SSH access to it yourself.
        '';
      };

      authorizedKeys = lib.mkOption {
        type = lib.types.listOf lib.types.singleLineStr;
        default = [ ];
        description = ''
          Public keys authorized to build on this machine, fed into
          `nix.sshServe.keys`. This should be the public half of the key(s)
          referenced by `discover.sshKeyPath` on the machines that are meant
          to dispatch builds here.
        '';
      };
    };

    discover = {
      enable = lib.mkEnableOption "nix-builder-beacon discover service";

      addBuilder = lib.mkEnableOption "nix-builder-beacon discover service" // {
        default = true;
      };

      outputPath = lib.mkOption {
        type = lib.types.path;
        default = "/var/lib/nix-builder-beacon/machines";
        description = "Path to the machines file to maintain.";
      };

      sshKeyPath = lib.mkOption {
        type = lib.types.path;
        description = ''
          Path to the SSH private key used to connect to discovered builders.
          This is a single shared cluster key: its public half must be listed
          in `sshServe.authorizedKeys` (or otherwise authorized) on every
          machine that should accept builds from here.

          The file must not be group- or other-readable, or `ssh` will refuse
          to use it. A raw Nix store path (mode 444) will *not* work; use
          e.g. `environment.etc.<name> = { source = ...; mode = "0600"; }`
          or a secrets-management tool (agenix, sops-nix, ...) to place it
          with the right permissions.
        '';
      };

      verbose = lib.mkOption {
        type = lib.types.bool;
        default = false;
        description = "Enable debug logs.";
      };
    };
  };

  config =
    let
      commonServiceConfig = {
        DynamicUser = true;
        Restart = "on-failure";
        RestartSec = "5s";
        NoNewPrivileges = true;
        PrivateTmp = true;
        PrivateDevices = true;
        PrivateUsers = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        ProtectProc = "invisible";
        ProtectHostname = true;
        ProtectClock = true;
        ProtectControlGroups = true;
        ProtectKernelLogs = true;
        ProtectKernelTunables = true;
        RestrictRealtime = true;
        CapabilityBoundingSet = "";
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_NETLINK"
          "AF_UNIX"
        ];
        RestrictNamespaces = true;
        LockPersonality = true;
        MemoryDenyWriteExecute = true;
        SystemCallFilter = [ "@system-service" ];
        SystemCallArchitectures = "native";
      };
    in
    lib.mkMerge [
      (lib.mkIf cfg.advert.enable {
        systemd.services.nix-builder-beacon-advert = {
          description = "nix-builder-beacon mDNS advertisement";
          wantedBy = [ "multi-user.target" ];
          after = [ "network.target" ];
          serviceConfig = commonServiceConfig // {
            ExecStart =
              let
                systemsArgs = lib.concatMapStringsSep " " (s: "--systems ${lib.escapeShellArg s}") cfg.advert.systems;
                supportedArgs = lib.concatMapStringsSep " " (
                  f: "--supported-features ${lib.escapeShellArg f}"
                ) cfg.advert.supportedFeatures;
                mandatoryArgs = lib.concatMapStringsSep " " (
                  f: "--mandatory-features ${lib.escapeShellArg f}"
                ) cfg.advert.mandatoryFeatures;
              in
              "${lib.getExe cfg.package} advert"
              + " --port ${toString cfg.advert.port}"
              + lib.optionalString (cfg.advert.hostname != null) " --hostname ${lib.escapeShellArg cfg.advert.hostname}"
              + " ${systemsArgs}"
              + " --max-jobs ${toString cfg.advert.maxJobs}"
              + " --speed-factor ${toString cfg.advert.speedFactor}"
              + " ${supportedArgs}"
              + " ${mandatoryArgs}"
              + " --ssh-user ${lib.escapeShellArg cfg.advert.sshUser}"
              + lib.optionalString (
                cfg.advert.hostKeyFile != null
              ) " --ssh-host-key-file ${lib.escapeShellArg cfg.advert.hostKeyFile}";
          };
        };
      })

      (lib.mkIf cfg.sshServe.enable {
        # nix-builder-beacon advertises ssh-ng:// URIs, which require nix-daemon
        # --stdio on the serving end rather than nix.sshServe's default legacy
        # `nix-store --serve` protocol.
        nix.sshServe = {
          enable = true;
          protocol = "ssh-ng";
          keys = cfg.sshServe.authorizedKeys;
        };
      })

      (lib.mkIf cfg.discover.enable {
        systemd.services.nix-builder-beacon-discover = {
          description = "nix-builder-beacon Nix remote builder discovery";
          wantedBy = [ "multi-user.target" ];
          after = [ "network.target" ];
          serviceConfig = commonServiceConfig // {
            StateDirectory = "nix-builder-beacon";
            ExecStart =
              "${lib.getExe cfg.package} discover"
              + " --output ${lib.escapeShellArg cfg.discover.outputPath}"
              + " --ssh-key-path ${lib.escapeShellArg cfg.discover.sshKeyPath}"
              + lib.optionalString cfg.discover.verbose " --verbose";
          };
        };
      })

      (lib.mkIf (cfg.discover.enable && cfg.discover.addBuilder) {
        # nix-remote-build.nix forces nix.settings.builders to null unless
        # this is set, which would otherwise conflict with our definition.
        nix.distributedBuilds = true;
        nix.settings.builders = "@${cfg.discover.outputPath}";
      })
    ];
}
