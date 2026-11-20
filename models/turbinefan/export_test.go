package turbinefan

// ThicknessRatio is the table's thickness ratio at radius r for the rotor sz, interpolated linearly between stations.
func ThicknessRatio(r float64, sz Size) float64 {
	stations := stationsFor(sz)
	for i := 1; i < len(stations); i++ {
		a, b := stations[i-1], stations[i]
		if r <= b.r {
			return a.ratio + (b.ratio-a.ratio)*(r-a.r)/(b.r-a.r)
		}
	}
	return stations[len(stations)-1].ratio
}

// StationRadiusChord returns the radius and chord of station k (1 to 10) of sz's blade table.
func StationRadiusChord(sz Size, k int) (float64, float64) {
	st := stationsFor(sz)[k-1]
	return st.r, st.chord
}

// SectionSurfacePoints returns station k's (1 to 10) pressure and suction surface points at chord fraction 0.4, in the
// right-hand blade's local section plane (X negated) at the reference pitch.
func SectionSurfacePoints(sz Size, k int) ([2]float64, [2]float64) {
	st := stationsFor(sz)[k-1]
	beta := sz.Beta(st.r, refPitch)
	mean := st.chord * camberLine(0.4, st.camber)
	half := halfThickness(0.4, st)
	pressure := sectionLocal(st, beta, 0.4, mean-half)
	suction := sectionLocal(st, beta, 0.4, mean+half)
	pressure[0], suction[0] = -pressure[0], -suction[0]
	return pressure, suction
}
