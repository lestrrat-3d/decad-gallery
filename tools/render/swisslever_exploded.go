package main

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/models/swisslever"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
)

// The exploded view is 1600 × 900, the smallest size docs/model-pipeline.md "Exploded views" allows.
const (
	swissExplodedWidth  = 1600
	swissExplodedHeight = 900
	// swissExplodedGap is the free space, in mm, between two layers of the exploded view.
	swissExplodedGap = 16.0
	// swissGuideMinSize is the longest box edge, in mm, a part needs to get a guide line in the exploded view. It
	// leaves the nuts, bolts and small bearings out, which would clutter the image.
	swissGuideMinSize = 13.0
)

// swissExplodedDir is the exploded view's camera direction: from the right, the front and a little above, so the
// layers stack from the back plate at the right to the knob at the left and every layer shows its face.
var swissExplodedDir = r3.NewVec(1, 0.3, 0.9)

// swissExplodedLayers lists the layers of the exploded view from the back to the front, which is also the assembly
// order the README's alt text follows. The three arbors stay where they are; every later layer is moved along +Z
// until it lies in a slab of its own, beyond the arbors' front ends, so no part hides on another's axis. Bodies of one
// layer sit on different axes and do not overlap in the XY plane.
var swissExplodedLayers = [][]string{
	{swisslever.EscapeArborName, swisslever.PalletArborName, swisslever.BalanceStaffName},
	{swisslever.BackEscapeBearingName, swisslever.BackPalletBearingName, swisslever.BackBalanceBearingName},
	{swisslever.BackPlateName},
	{swisslever.NutName(0), swisslever.NutName(1), swisslever.NutName(2), swisslever.NutName(3)},
	{swisslever.EntryStoneName, swisslever.SafetyRollerName},
	{swisslever.ExitStoneName},
	{swisslever.LeverName},
	{swisslever.EscapeWheelName, swisslever.RollerTableName},
	{swisslever.BalanceWheelName},
	{swisslever.HairspringName},
	{swisslever.HairspringRetainerName},
	{swisslever.FrontEscapeBearingName, swisslever.FrontPalletBearingName, swisslever.FrontBalanceBearingName},
	{swisslever.FrontPlateName},
	{swisslever.BoltName(0), swisslever.BoltName(1), swisslever.BoltName(2), swisslever.BoltName(3)},
	{swisslever.KnobCollarName},
	{swisslever.WindingKnobName},
}

// swissExplodedOffsets returns the offset of every body in the exploded view, by name. The first layer stays; each
// later layer moves along +Z until its rear face lies swissExplodedGap in front of the previous layer's front face.
func (b *swissBuild) swissExplodedOffsets() (map[string]r3.Vec, error) {
	offsets := map[string]r3.Vec{}
	var front float64
	for i, layer := range swissExplodedLayers {
		low, high := math.Inf(1), math.Inf(-1)
		for _, name := range layer {
			mesh, ok := b.meshes[name]
			if !ok {
				return nil, fmt.Errorf("exploded layer %d: no body %q", i, name)
			}
			for _, v := range mesh.Vertices() {
				low, high = math.Min(low, v.Z), math.Max(high, v.Z)
			}
		}
		shift := 0.0
		if i > 0 {
			shift = front + swissExplodedGap - low
		}
		for _, name := range layer {
			if _, dup := offsets[name]; dup {
				return nil, fmt.Errorf("exploded layers list %q twice", name)
			}
			offsets[name] = r3.NewVec(0, 0, shift)
		}
		front = high + shift
	}
	for _, nb := range b.named {
		if _, ok := offsets[nb.Name]; !ok {
			return nil, fmt.Errorf("exploded layers leave out %q", nb.Name)
		}
	}
	return offsets, nil
}

// explodedScene is the movement pulled apart along its depth, one slab per layer, with a dotted guide line from each
// larger part back to where it assembles.
func (b *swissBuild) explodedScene() (solidlens.Scene, error) {
	offsets, err := b.swissExplodedOffsets()
	if err != nil {
		return solidlens.Scene{}, err
	}
	models, err := b.models(everything, offsets)
	if err != nil {
		return solidlens.Scene{}, err
	}
	scene := bevelScene(swissCamera(models, swissExplodedDir, float64(swissExplodedWidth)/swissExplodedHeight), models)
	for _, nb := range b.named {
		offset := offsets[nb.Name]
		if offset == (r3.Vec{}) {
			continue
		}
		low, high := meshBounds(b.meshes[nb.Name])
		size := high.Sub(low)
		if math.Max(size.X, math.Max(size.Y, size.Z)) < swissGuideMinSize {
			continue
		}
		centre := low.Add(high).Scale(0.5)
		scene.Lines = append(scene.Lines, solidlens.Line{
			Points: []solidlens.Vec{solidlens.Vec(centre), solidlens.Vec(centre.Add(offset))},
			Color:  solidlens.RGB(0.1, 0.12, 0.16), Width: 1.5, Dash: []float64{6, 5}})
	}
	return scene, nil
}
