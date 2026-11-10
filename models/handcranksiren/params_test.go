package handcranksiren_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	require.NoError(t, handcranksiren.DefaultParams().Validate())
	for name, edit := range map[string]func(*handcranksiren.Params){
		"nan angle":      func(p *handcranksiren.Params) { p.CrankAngle = math.NaN() },
		"infinite angle": func(p *handcranksiren.Params) { p.CrankAngle = math.Inf(1) },
		"huge angle":     func(p *handcranksiren.Params) { p.CrankAngle = math.MaxFloat64 },
		"eleven slots":   func(p *handcranksiren.Params) { p.Slots = 11 },
		"25 slots":       func(p *handcranksiren.Params) { p.Slots = 25 },
	} {
		p := handcranksiren.DefaultParams()
		edit(&p)
		require.Error(t, p.Validate(), name)
		doc := handcranksiren.NewDocument()
		_, err := handcranksiren.Build(t.Context(), doc, p)
		require.Error(t, err, name)
		require.Empty(t, doc.Bodies(), "%s leaves the document unchanged", name)
	}
	for _, slots := range []int{12, 24} {
		p := handcranksiren.DefaultParams()
		p.Slots = slots
		require.NoError(t, p.Validate())
	}
}

func TestMotion(t *testing.T) {
	p := handcranksiren.DefaultParams()
	s, err := handcranksiren.Motion(p)
	require.NoError(t, err)
	require.InDelta(t, -3.125*p.CrankAngle, s.IntermediateAngle, 1e-12)
	require.InDelta(t, 10*p.CrankAngle, s.RotorAngle, 1e-12)
	require.InDelta(t, 28.18594, s.Intermediate.X, 1e-4)
	require.InDelta(t, 46.07751, s.Intermediate.Z, 1e-4)
	require.InDelta(t, 84, s.Rotor.Z, 0)
	require.InDelta(t, 60, s.Crank.X, 0)
	require.InDelta(t, 84, s.Crank.Z, 0)
	// The three axes keep the centre distances of the two stages.
	require.InDelta(t, 49.5, s.Intermediate.Sub(s.Crank).Len(), 1e-9)
	require.InDelta(t, 47.25, s.Intermediate.Sub(s.Rotor).Len(), 1e-9)

	s, err = handcranksiren.Motion(handcranksiren.Params{CrankAngle: math.Pi / 2, Slots: 20})
	require.NoError(t, err)
	require.InDelta(t, -1, s.ArmDirection.X, 1e-12, "a quarter turn points the arm toward -X")
	require.InDelta(t, 0, s.ArmDirection.Z, 1e-12)
	s, err = handcranksiren.Motion(handcranksiren.Params{CrankAngle: 0, Slots: 20})
	require.NoError(t, err)
	require.InDelta(t, 1, s.ArmDirection.Z, 1e-12, "angle 0 points the arm up")
}

func TestPitch(t *testing.T) {
	require.Equal(t, 400.0, handcranksiren.PitchHz(20, 2))
	require.Equal(t, 240.0, handcranksiren.PitchHz(12, 2))
	require.Equal(t, 480.0, handcranksiren.PitchHz(24, 2))
	require.InDelta(t, 50.0/16*48/15, handcranksiren.Ratio, 1e-12)
}

func TestBuildRejectsNilDocument(t *testing.T) {
	_, err := handcranksiren.Build(t.Context(), nil, handcranksiren.DefaultParams())
	require.Error(t, err)
}
