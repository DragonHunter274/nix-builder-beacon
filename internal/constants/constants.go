package constants

import (
	"github.com/betamos/zeroconf"
)

const MDNS_SERVICE = "_nix_remote_builder._tcp"

var ServiceType = zeroconf.NewType(MDNS_SERVICE)
