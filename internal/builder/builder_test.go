package builder

import (
	"testing"

	"github.com/betamos/zeroconf"
)

func svc(text []string) *zeroconf.Service {
	return &zeroconf.Service{
		Name:     "abc-123",
		Hostname: "server.local",
		Port:     22,
		Text:     text,
	}
}

func TestParseEvent_Minimal(t *testing.T) {
	b, err := ParseEvent(svc([]string{"systems=x86_64-linux"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if b.ID != "abc-123" || b.Hostname != "server.local" || b.Port != 22 {
		t.Errorf("unexpected identity fields: %+v", b)
	}
	if len(b.Systems) != 1 || b.Systems[0] != "x86_64-linux" {
		t.Errorf("unexpected systems: %v", b.Systems)
	}
	if b.SSHUser != DefaultSSHUser {
		t.Errorf("expected default ssh user, got %q", b.SSHUser)
	}
	if b.MaxJobs != DefaultMaxJobs || b.SpeedFactor != DefaultSpeedFactor {
		t.Errorf("unexpected defaults: maxJobs=%d speedFactor=%d", b.MaxJobs, b.SpeedFactor)
	}
	if b.HostKey != "" {
		t.Errorf("expected empty host key, got %q", b.HostKey)
	}
}

func TestParseEvent_MissingSystems(t *testing.T) {
	if _, err := ParseEvent(svc(nil)); err == nil {
		t.Fatal("expected error for missing systems field")
	}
}

func TestParseEvent_Full(t *testing.T) {
	b, err := ParseEvent(svc([]string{
		"systems=x86_64-linux,aarch64-linux",
		"maxJobs=4",
		"speedFactor=2",
		"supportedFeatures=kvm,big-parallel",
		"mandatoryFeatures=benchmark",
		"sshUser=builder",
		"hostKey=AAAAC3NzaC1lZDI1NTE5AAAA",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if b.MaxJobs != 4 || b.SpeedFactor != 2 {
		t.Errorf("unexpected job settings: %+v", b)
	}
	if b.SSHUser != "builder" {
		t.Errorf("unexpected ssh user: %q", b.SSHUser)
	}
	if b.HostKey != "AAAAC3NzaC1lZDI1NTE5AAAA" {
		t.Errorf("unexpected host key: %q", b.HostKey)
	}
	if len(b.SupportedFeatures) != 2 || len(b.MandatoryFeatures) != 1 {
		t.Errorf("unexpected features: %+v %+v", b.SupportedFeatures, b.MandatoryFeatures)
	}
}

func TestParseEvent_InvalidMaxJobs(t *testing.T) {
	if _, err := ParseEvent(svc([]string{"systems=x86_64-linux", "maxJobs=nope"})); err == nil {
		t.Fatal("expected error for invalid maxJobs")
	}
}

func TestMachinesLine_DefaultPort(t *testing.T) {
	b, err := ParseEvent(svc([]string{"systems=x86_64-linux"}))
	if err != nil {
		t.Fatal(err)
	}

	got := b.MachinesLine("/etc/nix-builder-beacon/key")
	want := "ssh-ng://nix-ssh@server.local x86_64-linux /etc/nix-builder-beacon/key 1 1 - - -"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestMachinesLine_NonDefaultPort(t *testing.T) {
	s := svc([]string{"systems=x86_64-linux"})
	s.Port = 2222
	b, err := ParseEvent(s)
	if err != nil {
		t.Fatal(err)
	}

	got := b.MachinesLine("/key")
	want := "ssh-ng://nix-ssh@server.local:2222 x86_64-linux /key 1 1 - - -"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestMachinesLine_FeaturesAndHostKey(t *testing.T) {
	b, err := ParseEvent(svc([]string{
		"systems=x86_64-linux",
		"supportedFeatures=kvm",
		"mandatoryFeatures=benchmark",
		"hostKey=AAAA",
	}))
	if err != nil {
		t.Fatal(err)
	}

	got := b.MachinesLine("/key")
	want := "ssh-ng://nix-ssh@server.local x86_64-linux /key 1 1 kvm benchmark AAAA"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestRender_Empty(t *testing.T) {
	if got := Render(nil, "/key"); len(got) != 0 {
		t.Errorf("expected empty output for no builders, got %q", got)
	}
}

func TestRender_Multiple(t *testing.T) {
	b1, _ := ParseEvent(svc([]string{"systems=x86_64-linux"}))
	b2, _ := ParseEvent(&zeroconf.Service{
		Name:     "def-456",
		Hostname: "other.local",
		Port:     22,
		Text:     []string{"systems=aarch64-linux"},
	})

	got := string(Render([]*Builder{b1, b2}, "/key"))
	want := "ssh-ng://nix-ssh@server.local x86_64-linux /key 1 1 - - -\n" +
		"ssh-ng://nix-ssh@other.local aarch64-linux /key 1 1 - - -\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}
