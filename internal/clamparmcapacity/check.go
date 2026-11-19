// Package clamparmcapacity checks the clamp arm's documented joint capacity.
package clamparmcapacity

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
)

// Inputs returns the mean friction radius and force spent closing a fin.
func Inputs() (float64, float64) {
	ro, ri := clamparm.EndRadius, m5fastener.HoleRadius
	rm := 2.0 / 3.0 * (ro*ro*ro - ri*ri*ri) / (ro*ro - ri*ri)
	const youngs = 3500.0
	second := clamparm.Section * clamparm.FinThickness * clamparm.FinThickness * clamparm.FinThickness / 12
	force := 3 * youngs * second * clamparm.JointFit / (clamparm.SlotDepth * clamparm.SlotDepth * clamparm.SlotDepth)
	return rm, force
}

// Check validates the three capacity rows in the Joint capacity section.
func Check(text string) error {
	var lines []string
	fence := byte(0)
	fenceLength := 0
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
			length := 0
			for length < len(trimmed) && trimmed[length] == trimmed[0] {
				length++
			}
			if length >= 3 {
				if fence == 0 {
					fence, fenceLength = trimmed[0], length
				} else if trimmed[0] == fence && length >= fenceLength && strings.TrimSpace(trimmed[length:]) == "" {
					fence = 0
				}
				lines = append(lines, "")
				continue
			}
		}
		if fence != 0 {
			lines = append(lines, "")
		} else {
			lines = append(lines, line)
		}
	}
	sectionStart, sectionEnd := -1, len(lines)
	for i, line := range lines {
		if line == "## Joint capacity" {
			if sectionStart >= 0 {
				return fmt.Errorf("expected exactly one Joint capacity section")
			}
			sectionStart = i
		} else if sectionStart >= 0 && sectionEnd == len(lines) && strings.HasPrefix(line, "## ") {
			sectionEnd = i
		}
	}
	if sectionStart < 0 {
		return fmt.Errorf("expected exactly one Joint capacity section")
	}
	const header = "| Inputs | `mu` | `T_k` (N.m) | `F` (N) | `T_hold` (N.m) |"
	tableStart := -1
	for i := sectionStart + 1; i < sectionEnd; i++ {
		if lines[i] == header {
			if tableStart >= 0 {
				return fmt.Errorf("expected exactly one capacity table header")
			}
			tableStart = i
		}
	}
	if tableStart < 0 {
		return fmt.Errorf("expected exactly one capacity table header")
	}
	if tableStart+1 >= sectionEnd || strings.TrimSpace(lines[tableStart+1]) != "| --- | --- | --- | --- | --- |" {
		return fmt.Errorf("invalid capacity table separator")
	}
	seen := make(map[string]bool)
	rm, fin := Inputs()
	for _, line := range lines[tableStart+2 : sectionEnd] {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			break
		}
		cells := strings.Split(strings.TrimSpace(line), "|")
		if len(cells) < 2 {
			continue
		}
		name := strings.TrimSpace(cells[1])
		if name != "Low" && name != "Nominal" && name != "High" {
			return fmt.Errorf("unexpected capacity row %q", name)
		}
		if seen[name] || len(cells) != 7 || strings.TrimSpace(cells[0]) != "" || strings.TrimSpace(cells[6]) != "" {
			return fmt.Errorf("%s: duplicate or malformed row", name)
		}
		seen[name] = true
		var values [4]float64
		for i := range values {
			value, err := strconv.ParseFloat(strings.TrimSpace(cells[i+2]), 64)
			if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
				return fmt.Errorf("%s: invalid number in column %d", name, i+2)
			}
			values[i] = value
		}
		mu, torque, force, hold := values[0], values[1], values[2], values[3]
		if math.Abs(force-torque/(0.2*5)*1000) > 1e-9 {
			return fmt.Errorf("%s: F does not match tightening torque", name)
		}
		if math.Abs(hold-2*mu*(force-fin)*rm/1000) > 0.01 {
			return fmt.Errorf("%s: T_hold does not match the model dimensions", name)
		}
	}
	if len(seen) != 3 {
		return fmt.Errorf("expected Low, Nominal and High capacity rows")
	}
	return nil
}
