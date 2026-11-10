package planetarygearbox_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/stretchr/testify/require"
)

// requireContainers checks that every body of box sits in the container for its kind, that the root holds only
// those containers, and that the document passes container verification.
func requireContainers(t *testing.T, doc *decad.Document, box planetarygearbox.Gearbox) {
	t.Helper()
	want := map[string][]*decad.Body{
		planetarygearbox.HousingContainerName:       {box.Ring, box.FrontCover, box.RearCover},
		planetarygearbox.CoverBearingsContainerName: {box.FrontBearing, box.RearBearing},
		planetarygearbox.BoltsContainerName:         box.Bolts,
		planetarygearbox.NutsContainerName:          box.Nuts,
		planetarygearbox.ShaftsContainerName:        {box.InputShaft},
	}
	for _, s := range box.Stages {
		add := func(name string, bodies ...*decad.Body) { want[name] = append(want[name], bodies...) }
		add(planetarygearbox.SunGearsContainerName, s.Sun)
		add(planetarygearbox.PlanetGearsContainerName, s.Planets...)
		add(planetarygearbox.PlanetBearingsContainerName, s.Bearings...)
		add(planetarygearbox.CarrierPinsContainerName, s.Pins...)
		add(planetarygearbox.CarriersContainerName, s.Carrier)
		add(planetarygearbox.ShaftsContainerName, s.Shaft)
	}

	var names []string
	for _, c := range doc.Root().Children() {
		names = append(names, c.Name())
	}
	require.ElementsMatch(t, []string{
		planetarygearbox.HousingContainerName, planetarygearbox.CoverBearingsContainerName,
		planetarygearbox.BoltsContainerName, planetarygearbox.NutsContainerName,
		planetarygearbox.ShaftsContainerName, planetarygearbox.SunGearsContainerName,
		planetarygearbox.PlanetGearsContainerName, planetarygearbox.PlanetBearingsContainerName,
		planetarygearbox.CarriersContainerName, planetarygearbox.CarrierPinsContainerName,
	}, names)

	total := 0
	for name, bodies := range want {
		for _, b := range bodies {
			c, err := doc.ContainerOf(b)
			require.NoError(t, err)
			require.Equal(t, name, c.Name())
		}
		total += len(bodies)
	}
	require.Len(t, doc.Bodies(), total)
	require.NoError(t, doc.VerifyContainers().Err())
}

func TestBodyContainers(t *testing.T) {
	doc, box := defaultGearbox(t)
	require.Len(t, box.Stages, 1)
	requireContainers(t, doc, box)
}
