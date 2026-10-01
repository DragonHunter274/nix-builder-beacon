package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/betamos/zeroconf"
	"github.com/urfave/cli/v2"

	"github.com/DragonHunter274/nix-builder-beacon/internal/builder"
	"github.com/DragonHunter274/nix-builder-beacon/internal/constants"
)

func runDiscover(cliCtx *cli.Context) error {
	if cliCtx.Bool("verbose") {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	output := cliCtx.Path("output")
	sshKeyPath := cliCtx.Path("ssh-key-path")

	// Exit on abort
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	index := builder.NewIndex()

	writeMachinesFile := func() {
		data := builder.Render(index.Snapshot(), sshKeyPath)
		if err := builder.WriteFileAtomic(output, data); err != nil {
			slog.Error("failed to write machines file", "path", output, "err", err)
		}
	}

	client, err := zeroconf.New().
		Browse(func(event zeroconf.Event) {
			if !event.Service.Type.Equal(constants.ServiceType) {
				return
			}

			slog.Debug("received event", "event", event.String())

			switch event.Op {
			case zeroconf.OpRemoved:
				index.Remove(event.Name)
				slog.Info("removing", "id", event.Name, "hostname", event.Hostname)
			case zeroconf.OpAdded, zeroconf.OpUpdated:
				b, err := builder.ParseEvent(event.Service)
				if err != nil {
					slog.Warn("ignoring builder advert with invalid metadata", "id", event.Name, "hostname", event.Hostname, "err", err)
					return
				}

				slog.Info("adding", "id", b.ID, "hostname", b.Hostname, "systems", b.Systems, "maxJobs", b.MaxJobs)
				index.Add(b)
			}

			writeMachinesFile()
		}, constants.ServiceType).
		Open()
	if err != nil {
		return err
	}
	defer client.Close() // Don't forget to close, to notify others that we're going away

	// Watch network interfaces for changes & reannounce on change
	go func() {
		if err := watchInterfaces(ctx, func() {
			slog.Debug("change in network interface, reload client")
			client.Reload()
		}); err != nil {
			slog.Error("error watching interfaces", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	return nil
}
