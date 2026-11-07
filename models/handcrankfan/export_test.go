package handcrankfan

// Beta exposes the blade angle for the external tests.
var Beta = beta

// ThicknessRatio is the table's thickness ratio at radius r, interpolated linearly between stations.
func ThicknessRatio(r float64) float64 {
	for i := 1; i < len(stations); i++ {
		a, b := stations[i-1], stations[i]
		if r <= b.r {
			return a.ratio + (b.ratio-a.ratio)*(r-a.r)/(b.r-a.r)
		}
	}
	return stations[len(stations)-1].ratio
}
