package main

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/betamos/zeroconf"
	"github.com/gofrs/uuid/v5"
	"github.com/urfave/cli/v2"

	"github.com/DragonHunter274/nix-builder-beacon/internal/constants"
)

func runAdvert(ctx *cli.Context) error {
	hostname := ctx.String("hostname")
	if hostname == "" {
		localHostname, err := os.Hostname()
		if err != nil {
			return err
		}
		hostname = localHostname
	}

	// Qualify unqualified hostnames with the mDNS domain (e.g. "nixos" -> "nixos.local").
	// Otherwise the advertised SRV target is not resolvable.
	if !strings.Contains(hostname, ".") {
		hostname += "." + constants.ServiceType.Domain
	}

	port := ctx.Int("port")
	systems := ctx.StringSlice("systems")

	text := []string{
		"systems=" + strings.Join(systems, ","),
		"maxJobs=" + strconv.Itoa(ctx.Int("max-jobs")),
		"speedFactor=" + strconv.Itoa(ctx.Int("speed-factor")),
		"sshUser=" + ctx.String("ssh-user"),
	}

	if features := ctx.StringSlice("supported-features"); len(features) > 0 {
		text = append(text, "supportedFeatures="+strings.Join(features, ","))
	}
	if features := ctx.StringSlice("mandatory-features"); len(features) > 0 {
		text = append(text, "mandatoryFeatures="+strings.Join(features, ","))
	}

	if hostKeyFile := ctx.Path("ssh-host-key-file"); hostKeyFile != "" {
		data, err := os.ReadFile(hostKeyFile)
		if err != nil {
			return fmt.Errorf("reading ssh host key file: %w", err)
		}

		hostKey, err := encodeHostKeyTxt(data)
		if err != nil {
			return fmt.Errorf("ssh host key file %q: %w", hostKeyFile, err)
		}

		text = append(text, "hostKey="+hostKey)
	}

	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	name := id.String()

	svc := zeroconf.Service{
		Type:     constants.ServiceType,
		Name:     name,
		Port:     uint16(port),
		Hostname: hostname,
		Text:     text,
	}

	server, err := zeroconf.New().Publish(&svc).Open()
	if err != nil {
		return err
	}
	defer server.Close()

	slog.Info("started", "id", name, "topic", constants.MDNS_SERVICE, "hostname", hostname, "port", port, "systems", systems)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	return nil
}

// encodeHostKeyTxt builds the mDNS TXT "hostKey" value from the contents of
// an SSH public key file (e.g. ssh_host_ed25519_key.pub). Nix's machines-file
// / ssh-ng "base64-ssh-public-host-key" field is base64-decoded and written
// verbatim as a known_hosts line ("host <decoded>"), so the encoded value
// must be the whole "algorithm base64key" pair, not just the bare key blob.
func encodeHostKeyTxt(pubKeyFile []byte) (string, error) {
	fields := strings.Fields(string(pubKeyFile))
	if len(fields) < 2 {
		return "", fmt.Errorf("doesn't look like a public key")
	}

	knownHostsKey := fields[0] + " " + fields[1]
	return base64.StdEncoding.EncodeToString([]byte(knownHostsKey)), nil
}
