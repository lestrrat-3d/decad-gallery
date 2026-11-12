package handcrankfan

// Part fit classes in mm, from docs/model-pipeline.md "Part clearances": Tight 0.2, Regular 0.35, Loose 0.5. Each
// nominal gap is the class minimum plus fitMargin, so the measured lower bound clears the class. instructions.md lists
// the class of every fit. A press fit is not a class: the rules set it at 0.05 mm with no margin.
const (
	fitMargin  = 0.05             // added to each class minimum so the measured lower bound clears it
	bevelBound = 0.1              // the bevel bodies' held mesh bound, charged on fits that involve a bevel gear
	pressFit   = 0.05             // radial clearance of a bearing seat and of a bore pressed onto a shaft
	tightFit   = 0.2 + fitMargin  // 0.25
	regularFit = 0.35 + fitMargin // 0.40
	looseFit   = 0.5 + fitMargin  // 0.55
)
