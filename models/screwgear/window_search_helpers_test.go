package screwgear_test

import (
	"sync"

	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
)

var defaultWindows = sync.OnceValues(func() ([2]screwgear.WindowForTest, error) {
	return screwgear.WindowsForTest(screwgear.DefaultParams())
})
