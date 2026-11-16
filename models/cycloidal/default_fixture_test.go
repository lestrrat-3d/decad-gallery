package cycloidal_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

var defaultDriveFixture struct {
	once  sync.Once
	doc   *decad.Document
	drive *cycloidal.Drive
	err   error
}

func defaultFixture(ctx context.Context, p cycloidal.Params) (*decad.Document, *cycloidal.Drive, error) {
	if p != cycloidal.DefaultParams() {
		return nil, nil, errors.New("default fixture requires exact default parameters")
	}
	defaultDriveFixture.once.Do(func() {
		defaultDriveFixture.doc = chainDocument()
		drive, err := cycloidal.Build(ctx, defaultDriveFixture.doc, p)
		defaultDriveFixture.drive, defaultDriveFixture.err = &drive, err
	})
	return defaultDriveFixture.doc, defaultDriveFixture.drive, defaultDriveFixture.err
}

func TestDefaultFixture(t *testing.T) {
	p := cycloidal.DefaultParams()
	doc, drive, err := defaultFixture(t.Context(), p)
	require.NoError(t, err)
	againDoc, againDrive, err := defaultFixture(t.Context(), p)
	require.NoError(t, err)
	require.True(t, doc == againDoc)
	require.True(t, drive == againDrive)

	changes := map[string]func(*cycloidal.Params){
		"PinCount":                func(p *cycloidal.Params) { p.PinCount++ },
		"PinCircleDiameter":       func(p *cycloidal.Params) { p.PinCircleDiameter++ },
		"PinDiameter":             func(p *cycloidal.Params) { p.PinDiameter++ },
		"Eccentricity":            func(p *cycloidal.Params) { p.Eccentricity++ },
		"DiskClearance":           func(p *cycloidal.Params) { p.DiskClearance++ },
		"DiscCount":               func(p *cycloidal.Params) { p.DiscCount++ },
		"DiscThickness":           func(p *cycloidal.Params) { p.DiscThickness++ },
		"DiscGap":                 func(p *cycloidal.Params) { p.DiscGap++ },
		"CenterBearingDiameter":   func(p *cycloidal.Params) { p.CenterBearingDiameter++ },
		"InputShaftDiameter":      func(p *cycloidal.Params) { p.InputShaftDiameter++ },
		"BearingClearance":        func(p *cycloidal.Params) { p.BearingClearance++ },
		"OutputPinCircleDiameter": func(p *cycloidal.Params) { p.OutputPinCircleDiameter++ },
		"OutputPinCount":          func(p *cycloidal.Params) { p.OutputPinCount++ },
		"OutputPinDiameter":       func(p *cycloidal.Params) { p.OutputPinDiameter++ },
		"OutputPinClearance":      func(p *cycloidal.Params) { p.OutputPinClearance++ },
		"HousingWall":             func(p *cycloidal.Params) { p.HousingWall++ },
		"BaseThickness":           func(p *cycloidal.Params) { p.BaseThickness++ },
		"OutputPlateThickness":    func(p *cycloidal.Params) { p.OutputPlateThickness++ },
		"ChamferSize":             func(p *cycloidal.Params) { p.ChamferSize++ },
		"Placement":               func(p *cycloidal.Params) { p.Placement = &r3.Frame{} },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			changed := p
			change(&changed)
			doc, drive, err := defaultFixture(t.Context(), changed)
			require.Error(t, err)
			require.Nil(t, doc)
			require.Nil(t, drive)
		})
	}
}
