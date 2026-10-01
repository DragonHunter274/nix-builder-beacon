package builder

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/betamos/zeroconf"
)

const (
	DefaultMaxJobs     = 1
	DefaultSpeedFactor = 1
	DefaultSSHUser     = "nix-ssh"
	DefaultSSHPort     = 22
)

// Builder describes a Nix remote build machine discovered on the network.
type Builder struct {
	// mDNS instance name, unique per advertiser. Used as the index key.
	ID string

	Hostname string
	Port     uint16
	SSHUser  string

	Systems           []string
	MaxJobs           int
	SpeedFactor       int
	SupportedFeatures []string
	MandatoryFeatures []string

	// SSH host public key (column 2 of e.g. ssh_host_ed25519_key.pub), used to
	// pin host identity in the machines file instead of trust-on-first-use.
	// Empty if the advertiser didn't provide one.
	HostKey string
}

// ParseEvent builds a Builder from a zeroconf discovery event's service data.
func ParseEvent(svc *zeroconf.Service) (*Builder, error) {
	txt := make(map[string]string, len(svc.Text))
	for _, entry := range svc.Text {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		txt[key] = value
	}

	systemsRaw, ok := txt["systems"]
	if !ok || systemsRaw == "" {
		return nil, fmt.Errorf("missing required TXT field %q", "systems")
	}

	b := &Builder{
		ID:                svc.Name,
		Hostname:          svc.Hostname,
		Port:              svc.Port,
		SSHUser:           DefaultSSHUser,
		Systems:           splitCSV(systemsRaw),
		MaxJobs:           DefaultMaxJobs,
		SpeedFactor:       DefaultSpeedFactor,
		SupportedFeatures: splitCSV(txt["supportedFeatures"]),
		MandatoryFeatures: splitCSV(txt["mandatoryFeatures"]),
		HostKey:           txt["hostKey"],
	}

	if v, ok := txt["sshUser"]; ok && v != "" {
		b.SSHUser = v
	}

	if v, ok := txt["maxJobs"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid TXT field %q: %w", "maxJobs", err)
		}
		b.MaxJobs = n
	}

	if v, ok := txt["speedFactor"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid TXT field %q: %w", "speedFactor", err)
		}
		b.SpeedFactor = n
	}

	return b, nil
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func joinOrDash(parts []string) string {
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ",")
}

// MachinesLine renders this builder as one line of a Nix machines file, using
// sshKeyPath as the shared private key path for connecting to it.
//
// Format: storeUri systems sshKey maxJobs speedFactor supportedFeatures mandatoryFeatures hostKey
// See https://nix.dev/manual/nix/stable/advanced-topics/distributed-builds
func (b *Builder) MachinesLine(sshKeyPath string) string {
	// URI format: ssh-ng://[username@]hostname[:port]
	uri := fmt.Sprintf("ssh-ng://%s@%s", b.SSHUser, b.Hostname)
	if b.Port != DefaultSSHPort {
		uri += fmt.Sprintf(":%d", b.Port)
	}

	hostKey := b.HostKey
	if hostKey == "" {
		hostKey = "-"
	}

	return strings.Join([]string{
		uri,
		strings.Join(b.Systems, ","),
		sshKeyPath,
		strconv.Itoa(b.MaxJobs),
		strconv.Itoa(b.SpeedFactor),
		joinOrDash(b.SupportedFeatures),
		joinOrDash(b.MandatoryFeatures),
		hostKey,
	}, " ")
}
