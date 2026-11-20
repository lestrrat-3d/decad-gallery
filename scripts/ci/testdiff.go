//go:build ignore

// testdiff reports changed top-level tests when all other declarations are unchanged.
package main

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/build"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type snapshot struct {
	other []string
	tests map[string]string
}

var testName = regexp.MustCompile(`^Test($|[^a-z].*)`)

func read(path string) (*snapshot, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !bytes.Contains(src, []byte("// CI: isolated-tests")) {
		return nil, os.ErrInvalid
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return nil, err
	}
	s := &snapshot{tests: make(map[string]string)}
	// Build and compiler directives may affect every declaration.
	for _, line := range strings.Split(string(src), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "//go:") || strings.HasPrefix(line, "// +build ") {
			s.other = append(s.other, line)
		}
	}
	s.other = append(s.other, f.Name.Name)
	for _, d := range f.Decls {
		var out bytes.Buffer
		if err := format.Node(&out, fset, d); err != nil {
			return nil, err
		}
		fn, ok := d.(*ast.FuncDecl)
		if ok && fn.Name.Name == "TestMain" {
			return nil, os.ErrInvalid
		}
		if ok && fn.Recv == nil && testName.MatchString(fn.Name.Name) {
			callsTest := false
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				id, ok := node.(*ast.Ident)
				if ok && testName.MatchString(id.Name) {
					callsTest = true
				}
				return true
			})
			if callsTest {
				return nil, os.ErrInvalid
			}
			s.tests[fn.Name.Name] = out.String()
		} else {
			gen, ok := d.(*ast.GenDecl)
			if !ok || gen.Tok != token.IMPORT {
				return nil, os.ErrInvalid
			}
			s.other = append(s.other, out.String())
		}
	}
	return s, nil
}

// A predicate reads only its scalar parameters. Comparisons and boolean operators
// cannot call user code, modify state, or panic on values of these types.
func purePredicate(fn *ast.FuncDecl) bool {
	if fn.Recv != nil || fn.Type.TypeParams != nil || ast.IsExported(fn.Name.Name) ||
		fn.Body == nil || len(fn.Body.List) != 1 || fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	result := fn.Type.Results.List[0]
	typ, ok := result.Type.(*ast.Ident)
	if !ok || typ.Name != "bool" || len(result.Names) != 0 {
		return false
	}
	params := map[string]bool{}
	for _, p := range fn.Type.Params.List {
		id, ok := p.Type.(*ast.Ident)
		if !ok || len(p.Names) == 0 {
			return false
		}
		switch id.Name {
		case "bool", "float32", "float64", "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "string":
		default:
			return false
		}
		for _, name := range p.Names {
			params[name.Name] = true
		}
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	valid := true
	ast.Inspect(ret.Results[0], func(node ast.Node) bool {
		if !valid {
			return false
		}
		switch n := node.(type) {
		case nil, *ast.ParenExpr:
		case *ast.Ident:
			valid = params[n.Name]
		case *ast.BasicLit:
			valid = n.Kind == token.FLOAT || n.Kind == token.INT || n.Kind == token.STRING
		case *ast.UnaryExpr:
			valid = n.Op == token.NOT || n.Op == token.SUB || n.Op == token.ADD
		case *ast.BinaryExpr:
			switch n.Op {
			case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ, token.LAND, token.LOR:
			default:
				valid = false
			}
		default:
			valid = false
		}
		return valid
	})
	return valid
}

type predicate struct{ name, outside string }

func predicateRevision(path string) (predicate, bool) {
	src, err := os.ReadFile(path)
	if err != nil || bytes.Contains(src, []byte("//go:")) || bytes.Contains(src, []byte("//line ")) {
		return predicate{}, false
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return predicate{}, false
	}
	var result predicate
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Doc == nil {
			continue
		}
		for _, c := range fn.Doc.List {
			if c.Text != "// CI: isolated-predicate" {
				continue
			}
			if result.name != "" || !purePredicate(fn) {
				return predicate{}, false
			}
			start, end := fset.PositionFor(fn.Body.Pos(), false).Offset, fset.PositionFor(fn.Body.End(), false).Offset
			result = predicate{fn.Name.Name, string(src[:start]) + "\x00" + string(src[end:])}
		}
	}
	return result, result.name != ""
}

// Resolve the active package against the Go compiler's export data. Reject a
// package with missing dependencies or unresolved declarations.
func checkedPackage(fset *token.FileSet, files []*ast.File) (*types.Info, bool) {
	imports := map[string]bool{}
	for _, f := range files {
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return nil, false
			}
			imports[path] = true
		}
	}
	args := []string{"list", "-deps", "-export", "-f", "{{.ImportPath}}\t{{.Export}}"}
	for name := range imports {
		args = append(args, name)
	}
	if len(imports) == 0 {
		return nil, false
	}
	out, err := exec.Command("go", args...).Output()
	if err != nil {
		return nil, false
	}
	exports := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		name, path, ok := strings.Cut(line, "\t")
		if ok {
			exports[name] = path
		}
	}
	lookup := func(path string) (io.ReadCloser, error) { return os.Open(exports[path]) }
	config := types.Config{Importer: importer.ForCompiler(fset, "gc", lookup)}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	_, err = config.Check(files[0].Name.Name, fset, files, info)
	return info, err == nil
}

// A changed boolean may only enable a diagnostic on the current testing.T.
// Evaluate arguments to the predicate regardless of its return value. Only
// allow diagnostic operands that cannot call formatting methods or change state.
func diagnosticGuard(stmt *ast.IfStmt, info *types.Info) *ast.CallExpr {
	cond, ok := stmt.Cond.(*ast.CallExpr)
	if !ok || stmt.Else != nil || len(stmt.Body.List) != 1 {
		return nil
	}
	expr, ok := stmt.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return nil
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok || call.Ellipsis.IsValid() {
		return nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Errorf" {
		return nil
	}
	target, ok := sel.X.(*ast.Ident)
	if !ok || target.Obj == nil || types.TypeString(info.TypeOf(target), nil) != "*testing.T" {
		return nil
	}
	if _, ok := target.Obj.Decl.(*ast.Field); !ok {
		return nil
	}
	for _, arg := range call.Args {
		if _, ok := info.TypeOf(arg).(*types.Basic); !ok {
			return nil
		}
		for {
			field, ok := arg.(*ast.SelectorExpr)
			if !ok {
				break
			}
			selection := info.Selections[field]
			if selection == nil || selection.Kind() != types.FieldVal || selection.Indirect() {
				return nil
			}
			arg = field.X
		}
		switch arg.(type) {
		case *ast.BasicLit, *ast.Ident:
		default:
			return nil
		}
	}
	return cond
}

// Inspect every declaration in the package, including package initializers. Any
// occurrence outside a direct call in a private test helper or isolated test
// prevents narrowing. Spelling collisions can only add work or reject a plan.
func predicateTests(dir, filename, before, after string) []string {
	old, ok := predicateRevision(before)
	now, okNew := predicateRevision(after)
	if !ok || !okNew || old != now {
		return nil
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	// A made-up OS and architecture also reject filenames implicitly constrained
	// to the host platform, even when those files happen to match today's runner.
	ctx := build.Default
	ctx.GOOS, ctx.GOARCH = "unknown", "unknown"
	type declaration struct {
		node ast.Decl
		file string
		pkg  string
	}
	var decls []declaration
	count := map[string]int{}
	isTest := map[string]bool{}
	helperPackage := ""
	fset := token.NewFileSet()
	packageFiles := map[string][]*ast.File{}
	for _, entry := range files {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		switch filepath.Ext(name) {
		case ".md":
			continue
		case ".go":
		default:
			return nil
		}
		match, err := ctx.MatchFile(dir, name)
		path := filepath.Join(dir, name)
		src, readErr := os.ReadFile(path)
		if err != nil || !match || readErr != nil || bytes.Contains(src, []byte("//go:")) ||
			bytes.Contains(src, []byte("// +build")) || bytes.Contains(src, []byte("//line ")) {
			return nil
		}
		f, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			return nil
		}
		packageFiles[f.Name.Name] = append(packageFiles[f.Name.Name], f)
		if name == filename {
			helperPackage = f.Name.Name
		}
		for _, imp := range f.Imports {
			if imp.Path.Value == `"C"` {
				return nil
			}
		}
		for _, d := range f.Decls {
			decls = append(decls, declaration{d, name, f.Name.Name})
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Name.Name == "TestMain" || fn.Body == nil {
				return nil
			}
			if fn.Recv != nil {
				continue
			}
			count[fn.Name.Name]++
			if testName.MatchString(fn.Name.Name) {
				if _, err := read(path); err != nil {
					return nil
				}
				isTest[fn.Name.Name] = true
			}
		}
	}
	if helperPackage == "" {
		return nil
	}
	info, ok := checkedPackage(fset, packageFiles[helperPackage])
	if !ok {
		return nil
	}
	pending := []string{now.name}
	seen := map[string]bool{}
	var tests []string
	for len(pending) > 0 {
		name := pending[0]
		pending = pending[1:]
		if seen[name] {
			continue
		}
		seen[name] = true
		foundCaller := false
		if count[name] != 1 {
			return nil
		}
		for _, d := range decls {
			allowed := map[*ast.Ident]bool{}
			fn, hasFn := d.node.(*ast.FuncDecl)
			if hasFn && fn.Name.Name == name {
				if name == now.name {
					if fn.Recv != nil || d.pkg != helperPackage {
						return nil
					}
					for _, field := range append(append([]*ast.Field{}, fn.Type.Params.List...), fn.Type.Results.List...) {
						id, ok := field.Type.(*ast.Ident)
						if !ok || info.Uses[id] != types.Universe.Lookup(id.Name) {
							return nil
						}
					}
				}
				allowed[fn.Name] = true
			}
			called, invalid := false, false
			guards := map[*ast.CallExpr]bool{}
			subtests := map[*ast.FuncLit]bool{}
			ast.Inspect(d.node, func(n ast.Node) bool {
				switch n.(type) {
				case *ast.GoStmt, *ast.DeferStmt:
					ast.Inspect(n, func(child ast.Node) bool {
						if id, ok := child.(*ast.Ident); ok && id.Name == name {
							invalid = true
						}
						return true
					})
					return false
				}
				if lit, ok := n.(*ast.FuncLit); ok && !subtests[lit] {
					ast.Inspect(lit, func(n ast.Node) bool {
						if id, ok := n.(*ast.Ident); ok && id.Name == name {
							invalid = true
						}
						return true
					})
					return false
				}
				if stmt, ok := n.(*ast.IfStmt); ok && name == now.name && d.pkg == helperPackage {
					if cond := diagnosticGuard(stmt, info); cond != nil {
						guards[cond] = true
					}
				}
				if c, ok := n.(*ast.CallExpr); ok {
					if d.pkg == helperPackage && len(c.Args) == 2 {
						if method, ok := c.Fun.(*ast.SelectorExpr); ok && method.Sel.Name == "Run" && types.TypeString(info.TypeOf(method.X), nil) == "*testing.T" {
							if sub, ok := c.Args[1].(*ast.FuncLit); ok {
								subtests[sub] = true
							}
						}
					}
					if id, ok := c.Fun.(*ast.Ident); ok && id.Name == name {
						if d.pkg != helperPackage {
							invalid = true
							return false
						}
						obj := info.Uses[id]
						if _, ok := obj.(*types.Func); !ok || obj.Pkg() == nil || obj.Parent() != obj.Pkg().Scope() {
							invalid = true
						}
						if name == now.name && !guards[c] {
							invalid = true
						}
						allowed[id] = true
						called = true
					}
				}
				if id, ok := n.(*ast.Ident); ok && id.Name == name && !allowed[id] {
					invalid = true
				}
				return true
			})
			if invalid {
				return nil
			}
			if !called {
				continue
			}
			if !hasFn || fn.Recv != nil || d.pkg != helperPackage || !strings.HasSuffix(d.file, "_test.go") ||
				fn.Name.Name == "init" || ast.IsExported(fn.Name.Name) && !isTest[fn.Name.Name] {
				return nil
			}
			if seen[fn.Name.Name] {
				return nil
			}
			foundCaller = true
			pending = append(pending, fn.Name.Name)
		}
		if isTest[name] {
			tests = append(tests, name)
		} else if !foundCaller {
			return nil
		}
	}
	sort.Strings(tests)
	return tests
}

func main() {
	if len(os.Args) == 6 && os.Args[1] == "--predicate" {
		json.NewEncoder(os.Stdout).Encode(predicateTests(os.Args[2], os.Args[3], os.Args[4], os.Args[5]))
		return
	}
	if len(os.Args) == 3 && os.Args[1] == "--layout" {
		src, err := os.ReadFile(os.Args[2])
		if err != nil {
			os.Exit(2)
		}
		if bytes.Contains(src, []byte("// CI: isolated-predicate")) {
			if _, ok := predicateRevision(os.Args[2]); !ok {
				os.Exit(2)
			}
		}
		if bytes.Contains(src, []byte("// CI: isolated-tests")) {
			if _, err := read(os.Args[2]); err != nil {
				os.Exit(2)
			}
			return
		}
		f, err := parser.ParseFile(token.NewFileSet(), os.Args[2], src, 0)
		if err != nil {
			os.Exit(2)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && fn.Name.Name != "TestMain" && testName.MatchString(fn.Name.Name) {
				if _, err := read(os.Args[2]); err != nil {
					os.Exit(2)
				}
				return
			}
		}
		return
	}
	if len(os.Args) != 3 {
		os.Exit(2)
	}
	before, err := read(os.Args[1])
	if err != nil {
		os.Exit(2)
	}
	after, err := read(os.Args[2])
	if err != nil {
		os.Exit(2)
	}
	if strings.Join(before.other, "\x00") != strings.Join(after.other, "\x00") {
		json.NewEncoder(os.Stdout).Encode(nil)
		return
	}
	var changed []string
	for name, old := range before.tests {
		if _, ok := after.tests[name]; !ok {
			json.NewEncoder(os.Stdout).Encode(nil)
			return
		}
		if after.tests[name] != old {
			changed = append(changed, name)
		}
	}
	for name := range after.tests {
		if _, ok := before.tests[name]; !ok {
			changed = append(changed, name)
		}
	}
	sort.Strings(changed)
	json.NewEncoder(os.Stdout).Encode(changed)
}
