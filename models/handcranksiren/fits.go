package handcranksiren

// Part fit classes in mm, from docs/model-pipeline.md "Part clearances": Tight 0.2, Regular 0.35, Loose 0.5. Each
// nominal gap is the class minimum plus fitMargin, so the measured lower bound clears the class. instructions.md lists
// the class of every fit.
const (
	fitMargin     = 0.05             // added to each class minimum so the measured lower bound clears it
	tightFit      = 0.2 + fitMargin  // 0.25
	regularFit    = 0.35 + fitMargin // 0.40
	looseFit      = 0.5 + fitMargin  // 0.55
	seatClearance = 0.05             // radial clearance of a bearing seat (the bearing-seat rule)
)
