package main

import (
	"encoding/base64"
	"testing"
)

// Regression test: Nix base64-decodes the hostKey field and writes the
// result verbatim as a known_hosts line, so the encoded value must be the
// whole "algorithm base64key" pair -- not just the bare key blob -- or host
// key verification fails with "Host key verification failed".
func TestEncodeHostKeyTxt(t *testing.T) {
	pubKeyFile := []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKyeev0FqGbwv3yfXTWNc+06A32YemwjrM5oDfELpXrH root@tothemoon\n")

	got, err := encodeHostKeyTxt(pubKeyFile)
	if err != nil {
		t.Fatalf("encodeHostKeyTxt: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("hostKey value isn't valid base64: %v", err)
	}

	want := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKyeev0FqGbwv3yfXTWNc+06A32YemwjrM5oDfELpXrH"
	if string(decoded) != want {
		t.Errorf("decoded hostKey = %q, want %q", decoded, want)
	}
}

func TestEncodeHostKeyTxtInvalid(t *testing.T) {
	if _, err := encodeHostKeyTxt([]byte("not-a-key")); err == nil {
		t.Error("expected error for malformed public key file, got nil")
	}
}
