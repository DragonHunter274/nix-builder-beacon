package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "nix-builder-beacon",
		Usage: "Nix remote builder discovery",
		Commands: []*cli.Command{
			{
				Name:  "advert",
				Usage: "Advertise this machine as a Nix remote builder on the network",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "port",
						Usage: "SSH port to advertise",
						Value: 22,
					},
					&cli.StringFlag{
						Name:  "hostname",
						Usage: "Hostname to advertise. Defaults to the local machine hostname.",
					},
					&cli.StringSliceFlag{
						Name:     "systems",
						Usage:    "Nix systems this machine can build for (repeatable, e.g. --systems x86_64-linux)",
						Required: true,
					},
					&cli.IntFlag{
						Name:  "max-jobs",
						Usage: "Maximum number of concurrent build jobs",
						Value: 1,
					},
					&cli.IntFlag{
						Name:  "speed-factor",
						Usage: "Relative speed factor of this machine",
						Value: 1,
					},
					&cli.StringSliceFlag{
						Name:  "supported-features",
						Usage: "Supported system features (repeatable)",
					},
					&cli.StringSliceFlag{
						Name:  "mandatory-features",
						Usage: "Mandatory system features (repeatable)",
					},
					&cli.StringFlag{
						Name:  "ssh-user",
						Usage: "SSH user to advertise for connecting to this builder",
						Value: "nix-ssh",
					},
					&cli.PathFlag{
						Name:  "ssh-host-key-file",
						Usage: "Path to this machine's SSH host public key file (e.g. /etc/ssh/ssh_host_ed25519_key.pub), advertised so discoverers can pin it",
					},
				},
				Action: runAdvert,
			},
			{
				Name:  "discover",
				Usage: "Discover Nix remote builders on the network and maintain a machines file",
				Flags: []cli.Flag{
					&cli.PathFlag{
						Name:     "output",
						Aliases:  []string{"o"},
						Usage:    "Path to the machines file to maintain",
						Required: true,
					},
					&cli.PathFlag{
						Name:     "ssh-key-path",
						Usage:    "Path to the SSH private key used to connect to discovered builders",
						Required: true,
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Print debug statements",
					},
				},
				Action: runDiscover,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
