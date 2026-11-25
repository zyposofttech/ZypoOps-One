//go:build linux

package zyposonosim

import "zyposoft.local/operations/products/simulator"

func Profile() simulator.Profile {
	return simulator.Profile{Product: "zyposono-sim", Lock: "ACTIVE_ACQUISITION", Healthy: "ACQUISITION_READY", Initial: "sono-test-1", Release: "sono-test-2"}
}
