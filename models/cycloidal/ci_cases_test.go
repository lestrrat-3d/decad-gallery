package cycloidal_test

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"
)

// ciDiscoverCases reports a table without building its geometry when CI requests discovery.
func ciDiscoverCases(t *testing.T, names []string) bool {
	t.Helper()
	if os.Getenv("CI_DISCOVER_CASES") != "1" {
		return false
	}
	seen := make(map[string]struct{}, len(names))
	validName := regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	for _, name := range names {
		if !validName.MatchString(name) {
			t.Fatalf("unsupported case name %q", name)
		}
		if _, exists := seen[name]; exists {
			t.Fatalf("duplicate case name %q", name)
		}
		seen[name] = struct{}{}
	}
	if len(names) == 0 {
		t.Fatal("empty case table")
	}
	data, err := json.Marshal(struct {
		Parent string   `json:"parent"`
		Names  []string `json:"names"`
	}{Parent: t.Name(), Names: names})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("CI_CASES %s\n", data)
	return true
}

type ciCase[T any] struct {
	name  string
	value T
}

func ciRunCases[T any](t *testing.T, cases []ciCase[T], run func(*testing.T, T)) {
	t.Helper()
	names := make([]string, 0, len(cases))
	for _, tc := range cases {
		names = append(names, tc.name)
	}
	if ciDiscoverCases(t, names) {
		return
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { run(t, tc.value) })
	}
}
