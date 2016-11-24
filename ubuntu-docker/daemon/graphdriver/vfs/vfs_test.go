// +build linux

package vfs

import (
	"testing"

	"github.com/docker/docker/daemon/graphdriver/graphtest"

	"github.com/docker/docker/pkg/reexec"
)

func shortSkip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping privileged test in short mode")
	}
}

// This avoids creating a new driver for each test if all tests are run
// Make sure to put new tests between TestVfsSetup and TestVfsTeardown
func TestVfsSetup(t *testing.T) {
	shortSkip(t)

	reexec.Init()

	graphtest.GetDriver(t, "vfs")
}

func TestVfsCreateEmpty(t *testing.T) {
	shortSkip(t)

	graphtest.DriverTestCreateEmpty(t, "vfs")
}

func TestVfsCreateBase(t *testing.T) {
	shortSkip(t)

	graphtest.DriverTestCreateBase(t, "vfs")
}

func TestVfsCreateSnap(t *testing.T) {
	shortSkip(t)

	graphtest.DriverTestCreateSnap(t, "vfs")
}

func TestVfsTeardown(t *testing.T) {
	shortSkip(t)

	graphtest.PutDriver(t)
}
