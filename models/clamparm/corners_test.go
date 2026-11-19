package clamparm_test

// CI: isolated-tests

import (
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/stretchr/testify/require"
)

// TestClampArmCorners runs the corner test on every printed part in its print orientation. Only the hex pocket
// corners of the four knobs, the two screw knobs and the adapter read a finding, each exempt, and no exemption may
// claim nothing. The frame, pad, swivel, links and head read none: their pockets' corners lie on a floor the section
// planes do not cut. The cases are the ones that change a printed part: the link length.
func TestClampArmCorners(t *testing.T) {
	for _, c := range asmCases(t) {
		switch c.name {
		case "default", "short", "long":
		default:
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, m := asmBuild(t, c)
			names, parts := asmPrinted(m)
			for i, p := range parts {
				var exempt []cornercheck.Exempt
				switch {
				case names[i] == "adapter":
					exempt = append(exempt, hexCorners(p, "adapter nut pocket corners", m5fastener.NutPocketRadius))
				case strings.HasPrefix(names[i], "knob"):
					exempt = append(exempt, hexCorners(p, names[i]+" nut pocket corners", m5fastener.NutPocketRadius))
				case names[i] == "screw knob" || names[i] == "stud knob":
					exempt = append(exempt, hexCorners(p, names[i]+" head pocket corners", m5fastener.PressPocketRadius))
				}
				fs, err := cornercheck.Check(t.Context(), p.Body, p.PrintUp, exempt)
				require.NoError(t, err, names[i])
				require.Empty(t, fs, "%s: %s", names[i], cornercheck.Report(fs, 8))
			}
		})
	}
}
