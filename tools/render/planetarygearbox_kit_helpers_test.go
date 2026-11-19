package main

func shotsOf(parts []planetaryPart) []string {
	var shots []string
	for _, p := range parts {
		shots = append(shots, p.shot)
	}
	return shots
}
