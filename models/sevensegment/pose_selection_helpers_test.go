package sevensegment_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// poseRecorder stands in for *testing.T and records a failure instead of stopping the test.
type poseRecorder struct {
	message string
	failed  bool
}

func (r *poseRecorder) Helper() {}

func (r *poseRecorder) Errorf(format string, args ...any) {
	r.message = strings.TrimSpace(strings.Join(append([]string{format}, anyStrings(args)...), " "))
}

func (r *poseRecorder) FailNow() { r.failed = true; panic(r) }

func anyStrings(args []any) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i], _ = a.(string)
	}
	return out
}

// poseUses is the poses one test reads from the shared build.
type poseUses struct{ digit0, placed bool }

// externalPoseUses returns, for each function of the package's external tests, which poses it reads, including through
// the functions it calls. A call matches by name, so a method or a function of the same name over-reports and never
// under-reports. A direct use of the build itself counts as every pose.
func externalPoseUses(t *testing.T) map[string]poseUses {
	t.Helper()
	files, err := filepath.Glob("*_test.go")
	require.NoError(t, err)
	fset := token.NewFileSet()
	refs := map[string]map[string]struct{}{}
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, 0)
		require.NoError(t, err)
		if f.Name.Name != "sevensegment_test" {
			continue
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			names := refs[fn.Name.Name]
			if names == nil {
				names = map[string]struct{}{}
				refs[fn.Name.Name] = names
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				// A name bound inside a function is a local variable, not a call of a package function.
				if id, ok := n.(*ast.Ident); ok && (id.Obj == nil || id.Obj.Kind == ast.Fun) {
					names[id.Name] = struct{}{}
				}
				return true
			})
		}
	}
	// The fixture's own functions are the sources, not part of the walk. The default display is built whatever the
	// poses are.
	delete(refs, "builtDefault")
	delete(refs, "builtDigit0")
	delete(refs, "builtPlaced")
	delete(refs, "buildPosed")
	delete(refs, "requirePose")
	out := map[string]poseUses{}
	for name := range refs {
		var uses poseUses
		seen := map[string]struct{}{}
		var walk func(string)
		walk = func(fn string) {
			if _, ok := seen[fn]; ok {
				return
			}
			seen[fn] = struct{}{}
			for ref := range refs[fn] {
				switch ref {
				case "builtDigit0":
					uses.digit0 = true
				case "builtPlaced":
					uses.placed = true
				case "posedBuild", "buildPosed":
					uses.digit0, uses.placed = true, true
				default:
					walk(ref)
				}
			}
		}
		walk(name)
		out[name] = uses
	}
	return out
}
