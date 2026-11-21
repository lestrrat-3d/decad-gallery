package sevensegment_test

import (
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
)

// rampsPerCam is instructions.md "Ramps per cam": From and Rising per segment. The box stands on its back wall, so each
// segment plays the role of its opposite (a and d, b and e, c and f swap) against the usual order.
var rampsPerCam = map[sevensegment.Segment][]struct {
	from   int
	rising bool
}{
	sevensegment.SegD: {{0, false}, {1, true}, {3, false}, {4, true}},
	sevensegment.SegE: {{4, false}, {6, true}},
	sevensegment.SegF: {{1, false}, {2, true}},
	sevensegment.SegA: {{0, false}, {1, true}, {3, false}, {4, true}, {6, false}, {7, true}},
	sevensegment.SegB: {{0, false}, {1, true}, {2, false}, {5, true}, {6, false}, {7, true}, {8, false}, {9, true}},
	sevensegment.SegC: {{0, false}, {3, true}, {6, false}, {7, true}},
	sevensegment.SegG: {{1, true}, {6, false}, {7, true}, {9, false}},
}
