package main

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
)

// registry returns every model the tool renders, in alphabetical order by name with the gallery last.
func registry() []renderkit.Model {
	return []renderkit.Model{
		ballbearingModel(),
		bevelgearModel(),
		clamparmModel(),
		crankhandleModel(),
		cycloidalModel(),
		dovetailModel(),
		escapementModel(),
		fastenerModel(),
		handcrankfanModel(),
		handcranksirenModel(),
		helicalgearModel(),
		herringbonegearModel(),
		onewayclutchModel(),
		pinringModel(),
		planetarygearboxModel(),
		plaquestandModel(),
		screwgearModel(),
		spinningwheelModel(),
		spinningwheelPreviewModel(),
		spiralspringModel(),
		spurgearModel(),
		swissleverModel(),
		turbinefanModel(),
		galleryModel(),
	}
}

// readmeShot adapts a function that writes a README image set into outputDir to a shot.
func readmeShot(render func(ctx context.Context, outputDir string) error) renderkit.ShotFunc {
	return func(ctx context.Context, opts renderkit.Options) error { return render(ctx, opts.Output) }
}

func ballbearingModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero":  {Width: 1600, Height: 900, Scene: bearing608Hero},
		"605zz": {Width: 1600, Height: 900, Scene: bearing605Hero},
	}
	return renderkit.Model{
		Name:    "ballbearing",
		Dir:     "models/ballbearing",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderBearingReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func bevelgearModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero":              {Width: 1600, Height: 900, Scene: bevelGearHero},
		"top":               {Width: bevelViewWidth, Height: bevelViewHeight, Scene: bevelGearTop},
		"low":               {Width: bevelViewWidth, Height: bevelViewHeight, Scene: bevelGearLow},
		"high":              {Width: bevelViewWidth, Height: bevelViewHeight, Scene: bevelGearHigh},
		"straight-top":      {Width: bevelViewWidth, Height: bevelViewHeight, Scene: bevelGearStraightTop},
		"straight-low":      {Width: bevelViewWidth, Height: bevelViewHeight, Scene: bevelGearStraightLow},
		"straight-high":     {Width: bevelViewWidth, Height: bevelViewHeight, Scene: bevelGearStraightHigh},
		"exploded-spiral":   {Width: bevelViewWidth, Height: bevelViewHeight, Scene: bevelGearExplodedSpiral},
		"exploded-straight": {Width: bevelViewWidth, Height: bevelViewHeight, Scene: bevelGearExplodedStraight},
	}
	return renderkit.Model{
		Name:    "bevelgear",
		Dir:     "models/bevelgear",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderBevelReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func clamparmModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero":              {Width: 1600, Height: 900, Scene: clampArmView("hero")},
		"side":              {Width: 1600, Height: 900, Scene: clampArmView("side")},
		"back":              {Width: 1600, Height: 900, Scene: clampArmView("back")},
		"top":               {Width: 1600, Height: 900, Scene: clampArmView("top")},
		"poses":             {Width: 1600, Height: 900, Scene: clampArmView("poses")},
		"exploded":          {Width: 1600, Height: 900, Scene: clampArmView("exploded")},
		"clamp-section":     {Width: 1600, Height: 900, Scene: clampArmView("clamp-section")},
		"joint-section":     {Width: 1600, Height: 900, Scene: clampArmView("joint-section")},
		"interface-section": {Width: 1600, Height: 900, Scene: clampArmView("interface-section")},
		"interface":         {Width: 1600, Height: 900, Scene: clampArmView("interface")},
		"part-frame":        {Width: 480, Height: 270, Scene: clampArmPartByShot("frame")},
		"part-pad":          {Width: 480, Height: 270, Scene: clampArmPartByShot("pad")},
		"part-swivel":       {Width: 480, Height: 270, Scene: clampArmPartByShot("swivel")},
		"part-link":         {Width: 480, Height: 270, Scene: clampArmPartByShot("link")},
		"part-head":         {Width: 480, Height: 270, Scene: clampArmPartByShot("head")},
		"part-adapter":      {Width: 480, Height: 270, Scene: clampArmPartByShot("adapter")},
		"part-knob":         {Width: 480, Height: 270, Scene: clampArmPartByShot("knob")},
		"part-screw-knob":   {Width: 480, Height: 270, Scene: clampArmPartByShot("screw-knob")},
		"part-bolt-25":      {Width: 480, Height: 270, Scene: clampArmPartByShot("bolt-25")},
		"part-bolt-35":      {Width: 480, Height: 270, Scene: clampArmPartByShot("bolt-35")},
		"part-bolt-40":      {Width: 480, Height: 270, Scene: clampArmPartByShot("bolt-40")},
		"part-bolt-70":      {Width: 480, Height: 270, Scene: clampArmPartByShot("bolt-70")},
		"part-nut":          {Width: 480, Height: 270, Scene: clampArmPartByShot("nut")},
	}
	return renderkit.Model{
		Name:    "clamparm",
		Dir:     "models/clamparm",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderClampArmReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func crankhandleModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero":          {Width: 1600, Height: 900, Scene: crankHandleHero},
		"back":          {Width: 1600, Height: 900, Scene: crankHandleViewScene("back")},
		"side":          {Width: 1600, Height: 900, Scene: crankHandleViewScene("side")},
		"sizes":         {Width: 1600, Height: 900, Scene: crankHandleViewScene("sizes")},
		"cutaway":       {Width: 1600, Height: 900, Scene: crankHandleViewScene("cutaway")},
		"exploded":      {Width: 1600, Height: 900, Scene: crankHandleViewScene("exploded")},
		"split":         {Width: 1600, Height: 900, Scene: crankHandleViewScene("split")},
		"print":         {Width: 1600, Height: 900, Scene: crankHandleViewScene("print")},
		"clamp":         {Width: 1600, Height: 900, Scene: crankHandleViewScene("clamp")},
		"clamp-nut":     {Width: 1600, Height: 900, Scene: crankHandleViewScene("clamp-nut")},
		"clamp-section": {Width: 1600, Height: 900, Scene: crankHandleViewScene("clamp-section")},
		"cross-section": {Width: 1600, Height: 900, Scene: crankHandleViewScene("cross-section")},
	}
	return renderkit.Model{
		Name:    "crankhandle",
		Dir:     "models/crankhandle",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderCrankHandleReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func cycloidalModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero":     {Width: 1600, Height: 900, Scene: cycloidalHero},
		"exploded": {Width: 1600, Height: 900, Scene: cycloidalExploded},
	}
	addCycloidalPresets(presets)
	return renderkit.Model{
		Name:    "cycloidal",
		Dir:     "models/cycloidal",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderCycloidalReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func dovetailModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero":         {Width: 1600, Height: 900, Scene: dovetailHero},
		"end":          {Width: 1600, Height: 900, Scene: dovetailViewScene("end")},
		"rear":         {Width: 1600, Height: 900, Scene: dovetailViewScene("rear")},
		"low":          {Width: 1600, Height: 900, Scene: dovetailViewScene("low")},
		"exploded":     {Width: 1600, Height: 900, Scene: dovetailViewScene("exploded")},
		"tail-block":   {Width: 480, Height: 270, Scene: dovetailViewScene("tail-block")},
		"socket-block": {Width: 480, Height: 270, Scene: dovetailViewScene("socket-block")},
	}
	return renderkit.Model{
		Name:    "dovetail",
		Dir:     "features/dovetail",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderDovetailReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func escapementModel() renderkit.Model {
	presets := map[string]renderkit.Preset{}
	addEscapementPresets(presets)
	return renderkit.Model{
		Name:    "escapement",
		Dir:     "models/escapement",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderEscapementReadme),
			"motion": escapementMotionShot,
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
			{Shot: "motion", Out: "images/motion.gif"},
		},
	}
}

func fastenerModel() renderkit.Model {
	presets := map[string]renderkit.Preset{}
	addFastenerPresets(presets)
	return renderkit.Model{
		Name:    "fastener",
		Dir:     "models/fastener",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderFastenerReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func handcrankfanModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero":     {Width: 1600, Height: 900, Scene: handCrankFanHero},
		"exploded": {Width: 1600, Height: 900, Scene: handCrankFanExploded},
	}
	addHandCrankFanViewPresets(presets)
	addHandCrankFanPartPresets(presets)
	return renderkit.Model{
		Name:    "handcrankfan",
		Dir:     "models/handcrankfan",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme":    readmeShot(renderHandCrankFanReadme),
			"animation": handCrankFanAnimationShot,
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
			{Shot: "animation", Out: "images/motion.gif"},
		},
	}
}

func handcranksirenModel() renderkit.Model {
	presets := map[string]renderkit.Preset{}
	addHandCrankSirenPresets(presets)
	return renderkit.Model{
		Name:    "handcranksiren",
		Dir:     "models/handcranksiren",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderHandCrankSirenReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func helicalgearModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero": {Width: 1600, Height: 900, Scene: helicalGearHero},
		"top":  {Width: 1600, Height: 900, Scene: helicalGearTop},
		"low":  {Width: 1600, Height: 900, Scene: helicalGearLow},
	}
	return renderkit.Model{
		Name:    "helicalgear",
		Dir:     "models/helicalgear",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": func(ctx context.Context, opts renderkit.Options) error {
				return renderkit.ReadmeFromHero(ctx, opts.Output, helicalGearHero, helicalCameras)
			},
		},
		Jobs: []renderkit.Job{
			{Shot: "hero", Out: "images/hero.png"},
			{Shot: "readme", Out: "images"},
		},
	}
}

func herringbonegearModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero": {Width: 1600, Height: 900, Scene: herringboneGearHero},
		"top":  {Width: 1600, Height: 900, Scene: herringboneGearTop},
		"low":  {Width: 1600, Height: 900, Scene: herringboneGearLow},
	}
	return renderkit.Model{
		Name:    "herringbonegear",
		Dir:     "models/herringbonegear",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": func(ctx context.Context, opts renderkit.Options) error {
				return renderkit.ReadmeFromHero(ctx, opts.Output, herringboneGearHero, herringboneCameras)
			},
		},
		Jobs: []renderkit.Job{
			{Shot: "hero", Out: "images/hero.png"},
			{Shot: "readme", Out: "images"},
		},
	}
}

func onewayclutchModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero":             {Width: 1600, Height: 900, Scene: oneWayClutchHero},
		"front":            {Width: 1600, Height: 900, Scene: oneWayClutchView("front")},
		"back":             {Width: 1600, Height: 900, Scene: oneWayClutchView("back")},
		"side":             {Width: 1600, Height: 900, Scene: oneWayClutchView("side")},
		"section":          {Width: 1600, Height: 900, Scene: oneWayClutchView("section")},
		"cutaway":          {Width: 1600, Height: 900, Scene: oneWayClutchView("cutaway")},
		"exploded":         {Width: 1600, Height: 900, Scene: oneWayClutchView("exploded")},
		"adapter":          {Width: 1600, Height: 900, Scene: oneWayClutchAdapterView("adapter")},
		"adapter-exploded": {Width: 1600, Height: 900, Scene: oneWayClutchAdapterView("adapter-exploded")},
		"directions":       {Width: 1600, Height: 900, Scene: oneWayClutchSideBySide(clutchDirectionParams)},
		"sizes":            {Width: 1600, Height: 900, Scene: oneWayClutchSideBySide(clutchSizeParams)},
	}
	return renderkit.Model{
		Name:    "onewayclutch",
		Dir:     "models/onewayclutch",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderOneWayClutchReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func pinringModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero": {Width: 1600, Height: 900, Scene: pinRingHero},
		"top":  {Width: 1600, Height: 900, Scene: pinRingViewScene("top")},
		"side": {Width: 1600, Height: 900, Scene: pinRingViewScene("side")},
		"low":  {Width: 1600, Height: 900, Scene: pinRingViewScene("low")},
	}
	return renderkit.Model{
		Name:    "pinring",
		Dir:     "models/pinring",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderPinRingReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func planetarygearboxModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero": {Width: 1600, Height: 900, Scene: planetaryGearboxHero},
	}
	addPlanetaryFastenerPresets(presets)
	addPlanetaryExplodedPreset(presets)
	addPlanetaryViewPresets(presets)
	addPlanetaryPartPresets(presets)
	return renderkit.Model{
		Name:    "planetarygearbox",
		Dir:     "models/planetarygearbox",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": planetaryReadmeShot,
			"motion": planetaryMotionShot,
		},
		AcceptsKit: true,
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
			{Shot: "readme", Kit: "planet605-shaft608", Out: "images/kit-planet605-shaft608"},
			{Shot: "readme", Kit: "planet608-shaft605", Out: "images/kit-planet608-shaft605"},
			{Shot: "readme", Kit: "planet608-shaft608", Out: "images/kit-planet608-shaft608"},
		},
	}
}

func plaquestandModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero":         {Width: 1600, Height: 900, Scene: plaqueStandHero},
		"frame":        {Width: 1600, Height: 900, Scene: plaqueStandViewScene("frame")},
		"joint":        {Width: 1600, Height: 900, Scene: plaqueStandViewScene("joint")},
		"side":         {Width: 1600, Height: 900, Scene: plaqueStandViewScene("side")},
		"back":         {Width: 1600, Height: 900, Scene: plaqueStandViewScene("back")},
		"exploded":     {Width: 1600, Height: 900, Scene: plaqueStandViewScene("exploded")},
		"print":        {Width: 1600, Height: 900, Scene: plaqueStandViewScene("print")},
		"sizes":        {Width: 1600, Height: 900, Scene: plaqueStandViewScene("sizes")},
		"part-bracket": {Width: 1600, Height: 900, Scene: plaqueStandViewScene("part-bracket")},
		"part-rail":    {Width: 1600, Height: 900, Scene: plaqueStandViewScene("part-rail")},
		"part-bolt":    {Width: 1600, Height: 900, Scene: plaqueStandViewScene("part-bolt")},
		"part-nut":     {Width: 1600, Height: 900, Scene: plaqueStandViewScene("part-nut")},
		"part-plaque":  {Width: 1600, Height: 900, Scene: plaqueStandViewScene("part-plaque")},
	}
	return renderkit.Model{
		Name:    "plaquestand",
		Dir:     "models/plaquestand",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderPlaqueStandReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func screwgearModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero":     {Width: 1600, Height: 900, Scene: screwGearView("hero")},
		"top":      {Width: 1600, Height: 900, Scene: screwGearView("top")},
		"low":      {Width: 1600, Height: 900, Scene: screwGearView("low")},
		"cutaway":  {Width: 1600, Height: 900, Scene: screwGearView("cutaway")},
		"exploded": {Width: 1600, Height: 900, Scene: screwGearView("exploded")},
	}
	return renderkit.Model{
		Name:    "screwgear",
		Dir:     "models/screwgear",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderScrewGearReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func spinningwheelModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero": {Width: 1600, Height: 900, Scene: spinningWheelView("hero")},
		"rear": {Width: 1600, Height: 900, Scene: spinningWheelView("rear")},
	}
	addSpinningWheelExplodedPreset(presets)
	return renderkit.Model{
		Name:    "spinningwheel",
		Dir:     "models/spinningwheel",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme":     readmeShot(renderSpinningWheelReadme),
			"front-back": sheetShot("front-back"),
			"layout":     sheetShot("layout"),
			"flyer":      sheetShot("flyer"),
			"brake":      sheetShot("brake"),
			"whorl":      sheetShot("whorl"),
			"animation":  spinningWheelAnimationShot,
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func spiralspringModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero":  {Width: 1600, Height: 900, Scene: spiralSpringHero},
		"top":   {Width: 1600, Height: 900, Scene: spiralSpringViewScene("top")},
		"low":   {Width: 1600, Height: 900, Scene: spiralSpringViewScene("low")},
		"root":  {Width: 1600, Height: 900, Scene: spiralSpringViewScene("root")},
		"sizes": {Width: 1600, Height: 900, Scene: spiralSpringViewScene("sizes")},
	}
	return renderkit.Model{
		Name:    "spiralspring",
		Dir:     "models/spiralspring",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderSpiralSpringReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

func spurgearModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero": {Width: 1600, Height: 900, Scene: spurGearHero},
		"top":  {Width: 1600, Height: 900, Scene: spurGearTop},
		"low":  {Width: 1600, Height: 900, Scene: spurGearLow},
	}
	return renderkit.Model{
		Name:    "spurgear",
		Dir:     "models/spurgear",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": func(ctx context.Context, opts renderkit.Options) error {
				return renderkit.ReadmeFromHero(ctx, opts.Output, spurGearHero, nil)
			},
		},
		Jobs: []renderkit.Job{
			{Shot: "hero", Out: "images/hero.png"},
			{Shot: "top", Out: "images/top.png"},
			{Shot: "low", Out: "images/low.png"},
			{Shot: "readme", Out: "images"},
		},
	}
}

func swissleverModel() renderkit.Model {
	presets := map[string]renderkit.Preset{}
	addSwissPresets(presets)
	return renderkit.Model{
		Name:    "swisslever",
		Dir:     "models/swisslever",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderSwissReadme),
			"motion": swissMotionShot,
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
			{Shot: "motion", Out: "images/motion.gif"},
		},
	}
}

func turbinefanModel() renderkit.Model {
	presets := map[string]renderkit.Preset{
		"hero":            {Width: 1600, Height: 900, Scene: turbineFanHero},
		"front":           {Width: 1600, Height: 900, Scene: turbineFanViewScene("front")},
		"back":            {Width: 1600, Height: 900, Scene: turbineFanViewScene("back")},
		"side":            {Width: 1600, Height: 900, Scene: turbineFanViewScene("side")},
		"setscrew":        {Width: 1600, Height: 900, Scene: turbineFanViewScene("setscrew")},
		"joint":           {Width: 1600, Height: 900, Scene: turbineFanViewScene("joint")},
		"exploded":        {Width: 1600, Height: 900, Scene: turbineFanViewScene("exploded")},
		"exploded-back":   {Width: 1600, Height: 900, Scene: turbineFanViewScene("exploded-back")},
		"sizes":           {Width: 1600, Height: 900, Scene: turbineFanViewScene("sizes")},
		"clutch":          {Width: 1600, Height: 900, Scene: turbineClutchViewScene("clutch")},
		"clutch-back":     {Width: 1600, Height: 900, Scene: turbineClutchViewScene("clutch-back")},
		"clutch-cutaway":  {Width: 1600, Height: 900, Scene: turbineClutchViewScene("clutch-cutaway")},
		"clutch-exploded": {Width: 1600, Height: 900, Scene: turbineClutchViewScene("clutch-exploded")},
	}
	return renderkit.Model{
		Name:    "turbinefan",
		Dir:     "models/turbinefan",
		Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": readmeShot(renderTurbineFanReadme),
		},
		Jobs: []renderkit.Job{
			{Shot: "readme", Out: "images"},
		},
	}
}

// spinningwheelPreviewModel renders the work-in-progress spinning wheel preview.
func spinningwheelPreviewModel() renderkit.Model {
	shot := func(name string) renderkit.ShotFunc {
		return func(ctx context.Context, opts renderkit.Options) error {
			return renderSpinningWheelPreview(ctx, name, opts.Output, opts.Width, opts.Height)
		}
	}
	return renderkit.Model{
		Name: "spinningwheel-preview",
		Dir:  "models/spinningwheel",
		Shots: map[string]renderkit.ShotFunc{
			"hero":   shot("hero"),
			"rear":   shot("rear"),
			"readme": shot("readme"),
		},
	}
}

// galleryModel renders the gallery hero image and the contact sheet of every model's hero scene.
func galleryModel() renderkit.Model {
	return renderkit.Model{
		Name: "gallery",
		Dir:  "docs",
		Shots: map[string]renderkit.ShotFunc{
			"hero":          galleryShot(false),
			"contact-sheet": galleryShot(true),
		},
		Jobs: []renderkit.Job{{Shot: "hero", Out: "images/gallery.png"}},
	}
}

// galleryShot renders the hero image, or the contact sheet when sheet is set. Both are 16:9.
func galleryShot(sheet bool) renderkit.ShotFunc {
	return func(ctx context.Context, opts renderkit.Options) error {
		width, height := opts.Width, opts.Height
		if width == 0 {
			width = galleryWidth
		}
		if height == 0 {
			height = width * galleryHeight / galleryWidth
		}
		if width <= 0 || height <= 0 || width*galleryHeight != height*galleryWidth {
			return fmt.Errorf("gallery image must have positive 16:9 dimensions, got %d × %d", width, height)
		}
		if sheet {
			return renderGallery(ctx, opts.Output, width, height)
		}
		scene, err := ornamentalGears(ctx)
		if err != nil {
			return fmt.Errorf("build gallery/hero scene: %w", err)
		}
		return renderGalleryHero(ctx, opts.Output, scene, width, height)
	}
}

// sheetShot renders one spinning wheel sheet.
func sheetShot(name string) renderkit.ShotFunc {
	return func(ctx context.Context, opts renderkit.Options) error {
		return runSpinningWheelSheet(ctx, name, opts.Output, opts.Width, opts.Height)
	}
}

func spinningWheelAnimationShot(ctx context.Context, opts renderkit.Options) error {
	return runSpinningWheelAnimation(ctx, opts.Output, opts.Width, opts.Height)
}

// handCrankFanAnimationShot writes the fan animation; -frames shortens it.
func handCrankFanAnimationShot(ctx context.Context, opts renderkit.Options) error {
	width, height, frames := opts.Width, opts.Height, opts.Frames
	if width == 0 {
		width = 1000
	}
	if height == 0 {
		height = 560
	}
	if frames <= 0 {
		frames = fanAnimationFrames
	}
	return renderHandCrankFanAnimation(ctx, opts.Output, width, height, frames)
}

// planetaryReadmeShot writes the standard README set, or the -kit kit's set when a kit is given.
func planetaryReadmeShot(ctx context.Context, opts renderkit.Options) error {
	if opts.Kit != "" {
		return renderPlanetaryKitReadme(ctx, opts.Output)
	}
	return renderPlanetaryGearboxReadme(ctx, opts.Output)
}

func planetaryMotionShot(ctx context.Context, opts renderkit.Options) error {
	width, height := opts.Width, opts.Height
	if width == 0 {
		width = planetaryMotionWidth
	}
	if height == 0 {
		height = planetaryMotionHeight
	}
	return renderPlanetaryMotion(ctx, opts.Output, width, height, planetaryMotionFrames)
}

func escapementMotionShot(ctx context.Context, opts renderkit.Options) error {
	width, height := opts.Width, opts.Height
	if width == 0 {
		width = escapementMotionWidth
	}
	if height == 0 {
		height = escapementMotionHeight
	}
	return renderEscapementMotion(ctx, opts.Output, width, height, escapementMotionFrames)
}

func swissMotionShot(ctx context.Context, opts renderkit.Options) error {
	width, height := opts.Width, opts.Height
	if width == 0 {
		width = swissMotionWidth
	}
	if height == 0 {
		height = swissMotionHeight
	}
	return renderSwissMotion(ctx, opts.Output, width, height, swissMotionFrames)
}
