package planetarygearbox_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
)

// TestCollarRings applies requireCollarRings to the default gearbox.
func TestCollarRings(t *testing.T) {
	_, box := defaultGearbox(t)
	requireCollarRings(t, box, planetarygearbox.DefaultParams())
}
