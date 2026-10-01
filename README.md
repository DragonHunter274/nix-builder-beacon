# nix-builder-beacon - mDNS discovery for Nix remote builders

_Status_: Alpha.

`nix-builder-beacon` uses mDNS service discovery to announce & find Nix remote build
machines on the local network, keeping a [Nix machines file](https://nix.dev/manual/nix/stable/advanced-topics/distributed-builds)
in sync so `nix-daemon` can dispatch distributed builds to whatever build machines are
currently online, turning your entire network of Nix nodes into a build farm.

## Security

`nix-builder-beacon` doesn't change the security model of remote builds. Builders are
reached over SSH, authenticated with a single shared keypair you provide (see below) -
`nix-builder-beacon` doesn't generate or distribute keys for you.

Discovery traffic (mDNS) is unencrypted & has potential privacy implications, meaning
nodes on the network can see what build machines exist and their advertised
capabilities. Host authenticity is still verified: each advertisement includes the
node's SSH host public key, which is written into the machines file so `nix-daemon`
pins it on connect instead of trusting-on-first-use.

If using the NixOS module, discovered builders are wired into `nix.sshServe`
automatically using the same cryptographic keys you configure once.

## Usage

The primary way to use `nix-builder-beacon` is via its NixOS module.

```nix
{ ... }:
{
  services.nix-builder-beacon = {
    # Advertise this machine as a build machine on the local network
    advert = {
      enable = true;
      systems = [ "x86_64-linux" ];
      maxJobs = 4;
    };

    # Allow discovered peers using this keypair to build here
    sshServe.authorizedKeys = [
      (builtins.readFile ./cluster_key.pub)
    ];

    # Discover build machines on the local network & keep a machines file in sync
    discover = {
      enable = true;
      sshKeyPath = "/run/secrets/cluster_key"; # e.g. provisioned via agenix/sops-nix
    };
  };
}
```

## License

- The application is licensed under `GPL-3.0-or-later`
- Nix expressions are licensed under `MIT`
