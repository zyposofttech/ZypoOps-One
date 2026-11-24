//go:build linux

package secondary
import "zyposoft.local/operations/products/simulator"
func Profile()simulator.Profile{return simulator.Profile{Product:"mock-secondary",Lock:"BATCH_IN_PROGRESS",Healthy:"WORKER_READY",Initial:"worker-test-4",Release:"worker-test-5"}}
