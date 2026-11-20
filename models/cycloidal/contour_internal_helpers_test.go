package cycloidal

// contourConfigs are drives the contour tests sweep.
var contourConfigs = map[string]func(p *Params){
	"default":       func(p *Params) {},
	"twelve pins":   func(p *Params) { p.PinCount = 12 },
	"odd pins":      func(p *Params) { p.PinCount = 13 },
	"big eccentric": func(p *Params) { p.Eccentricity = 2 },
	"clearance":     func(p *Params) { p.DiskClearance = 0.6 },
}
