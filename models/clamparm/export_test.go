package clamparm

// Test hooks: the external tests reach the unexported builders through these.
var (
	BuildLink   = buildLink
	BuildKnob   = buildKnob
	SwivelBlank = swivelBlank
	BuildSwivel = buildSwivel
	BuildHead   = buildHead
	ForkSite    = forkSite
)
