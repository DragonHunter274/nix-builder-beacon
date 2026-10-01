{ buildGoModule, lib }:
buildGoModule {
  pname = "nix-builder-beacon";
  version = "0.1";
  src = lib.cleanSource ./.;
  vendorHash = "sha256-dlVgfUG1a9wSGAXb62tjbsdwYuOaObgpHbNL1b+cO6I=";
  env.CGO_ENABLED = "1"; # Required for .local domain lookup via glibc NSS
  ldflags = [
    "-s"
    "-w"
  ];
  meta.mainProgram = "nix-builder-beacon";
}
