package clamparmcapacity_test

import (
	"os"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/clamparmcapacity"
	"github.com/stretchr/testify/require"
)

func documentation(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../models/clamparm/instructions.md")
	require.NoError(t, err)
	return string(data)
}

func TestDocumentation(t *testing.T) {
	require.NoError(t, clamparmcapacity.Check(documentation(t)))
}

func TestCheck(t *testing.T) {
	actual := documentation(t)
	cases := []struct {
		name, old, replacement, message string
	}{
		{"missing section", "## Joint capacity", "## Capacity", "section"},
		{"missing row", "| High |", "| Other |", "unexpected capacity row"},
		{"missing table header", "| Inputs | `mu` | `T_k` (N.m) | `F` (N) | `T_hold` (N.m) |", "prose", "header"},
		{"header in prose", "| Inputs | `mu` | `T_k` (N.m) | `F` (N) | `T_hold` (N.m) |", "prose | Inputs | `mu` | `T_k` (N.m) | `F` (N) | `T_hold` (N.m) |", "header"},
		{"fenced table", "| Inputs | `mu` | `T_k` (N.m) | `F` (N) | `T_hold` (N.m) |", "```markdown\n| Inputs | `mu` | `T_k` (N.m) | `F` (N) | `T_hold` (N.m) |", "header"},
		{"separated rows", "| Low | 0.2 | 0.4 | 400 | 1.00 |", "\n| Low | 0.2 | 0.4 | 400 | 1.00 |", "rows"},
		{"duplicate row", "| Low |", "| Nominal |", "duplicate"},
		{"malformed row", "| Low | 0.2 | 0.4 | 400 | 1.00 |", "| Low | 0.2 | 0.4 | 400 |", "malformed"},
		{"invalid number", "| Low | 0.2 |", "| Low | NaN |", "invalid number"},
		{"wrong force", "| Low | 0.2 | 0.4 | 400 |", "| Low | 0.2 | 0.4 | 401 |", "F does not"},
		{"wrong capacity", "| Low | 0.2 | 0.4 | 400 | 1.00 |", "| Low | 0.2 | 0.4 | 400 | 1.10 |", "T_hold"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Contains(t, actual, tc.old)
			err := clamparmcapacity.Check(strings.Replace(actual, tc.old, tc.replacement, 1))
			require.ErrorContains(t, err, tc.message)
		})
	}
}
