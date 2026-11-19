package spiralspring_test

import (
	"math"

	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
)

// rootRoom returns the C4 fillet's contact x4 on the spoke's underside, the C3 fillet's contact x3(e) for the
// derived e, and r_f, from the construction's own formulas.
func rootRoom(p spiralspring.Params, dm spiralspring.Dimensions) (x4, x3 float64) {
	a := 2*p.Thickness + dm.FilletRadius
	x4 = -math.Sqrt((dm.HubRadius+dm.FilletRadius)*(dm.HubRadius+dm.FilletRadius) - a*a)
	r := dm.FirstInnerRadius - dm.TaperOffset - dm.FilletRadius
	x3 = dm.CenterOffset + dm.TaperOffset - math.Sqrt(r*r-a*a)
	return x4, x3
}
