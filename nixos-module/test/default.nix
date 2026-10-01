{
  pkgs ? import <nixpkgs> { },
}:
let
  inherit (pkgs) lib;

  # A derivation that doesn't exist anywhere but as a `.drv`, so realising it
  # forces an actual build rather than being satisfied from an existing store
  # path.
  testDrv = pkgs.runCommand "nix-builder-beacon-marker" { } ''
    echo "built by $(hostname)" > $out
  '';
in
pkgs.testers.nixosTest {
  name = "nix-builder-beacon-test";

  nodes = {
    server = {
      imports = [
        ../.
      ];

      services.nix-builder-beacon.advert = {
        enable = true;
        systems = [ "x86_64-linux" ];
      };

      services.nix-builder-beacon.sshServe.authorizedKeys = [
        (lib.removeSuffix "\n" (builtins.readFile ./id_ed25519.pub))
      ];

      networking.firewall.enable = false;
    };

    client = {
      imports = [
        ../.
      ];

      # Resolve .local hostnames over mDNS
      services.avahi = {
        enable = true;
        nssmdns4 = true;
        ipv4 = true;
        ipv6 = true;
      };

      # A raw Nix store copy of the private key would be world-readable
      # (mode 444), which ssh refuses to use. environment.etc with an
      # explicit mode materializes a real file with the right permissions.
      environment.etc."nix-builder-beacon-test-key" = {
        source = ./id_ed25519;
        mode = "0600";
      };

      services.nix-builder-beacon.discover = {
        enable = true;
        sshKeyPath = "/etc/nix-builder-beacon-test-key";
      };

      # Make the marker derivation (but not its build output) known to the
      # client, so nix-daemon can dispatch building it without it already
      # being present locally.
      virtualisation.additionalPaths = [ testDrv.drvPath ];

      networking.firewall.enable = false;
    };
  };

  testScript = ''
    start_all()

    server.wait_for_unit("sshd.service")
    server.wait_for_unit("nix-builder-beacon-advert.service")

    client.wait_for_unit("nix-builder-beacon-discover.service")

    # Cross-host mDNS hostname resolution
    client.wait_until_succeeds("getent hosts server.local")

    # Wait for the machines file to be populated via mDNS discovery
    client.wait_until_succeeds("grep -q server.local /var/lib/nix-builder-beacon/machines")

    # Force a real remote build over the mDNS-discovered SSH machine entry
    client.succeed("nix-store --realise ${testDrv.drvPath} --max-jobs 0 >&2")
  '';
}
