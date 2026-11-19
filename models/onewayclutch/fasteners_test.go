package onewayclutch_test

// CI: isolated-tests

import (
	"testing"
)

func TestOneWayClutchFasteners(t *testing.T) {
	for name, p := range fastenerCases() {
		t.Run(name, func(t *testing.T) {
			m, _ := buildCase(t, p)
			for i, site := range m.Sites {
				requireSiteFastener(t, m, i, site)
				requireSitePocket(t, m, i, site)
				requireSiteWalls(t, m, i, site)
				requireSiteClear(t, m, i, site)
			}
			requireHubBoltWalls(t, m)
		})
	}
}
