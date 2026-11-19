//go:build ignore

// testdiff reports changed top-level tests when all other declarations are unchanged.
package main

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strings"
)

type snapshot struct {
	other []string
	tests map[string]string
}

var testName = regexp.MustCompile(`^Test[^a-z].*`)

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

func main() {
	if len(os.Args) == 3 && os.Args[1] == "--layout" {
		src, err := os.ReadFile(os.Args[2])
		if err != nil {
			os.Exit(2)
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
