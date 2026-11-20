package sevensegment

import (
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
)

// atY exempts the corners of the shaft at world Y y, within 0.06 mm.
func atY(reason string, y float64) cornercheck.Exempt {
	return cornercheck.Exempt{Reason: reason, Optional: true, Near: func(p r3.Vec) bool { return math.Abs(p.Y-y) < 0.06 }}
}
