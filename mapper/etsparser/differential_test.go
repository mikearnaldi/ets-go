package etsparser_test

// Differential tests: the vendored ETS parser must parse any pure-TypeScript
// file identically to the stock typescript-go parser. This is the regression
// net proving ETS grammar extensions never leak into TS behavior.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/etsmapper/etsparser"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	stockparser "github.com/microsoft/typescript-go/internal/parser"
)

func corpusRoots(t *testing.T) []string {
	t.Helper()
	if dir := os.Getenv("TSGO_REPO"); dir != "" {
		return []string{filepath.Join(dir, "testdata", "tests", "cases")}
	}
	return []string{filepath.Join("..", "..", "typescript-go", "testdata", "tests", "cases")}
}

func scriptKindFor(path string) core.ScriptKind {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".tsx":
		return core.ScriptKindTSX
	case ".js", ".jsx", ".mjs", ".cjs":
		return core.ScriptKindJSX
	default:
		return core.ScriptKindTS
	}
}

func collectCorpus(t *testing.T) []string {
	t.Helper()
	var files []string
	for _, root := range corpusRoots(t) {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			t.Fatal(err)
		}
		err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".ts", ".tsx", ".js", ".jsx", ".mts", ".cts":
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking corpus %s: %v", root, err)
		}
	}
	if len(files) == 0 {
		t.Fatal("corpus is empty")
	}
	return files
}

func TestDifferentialCorpus(t *testing.T) {
	files := collectCorpus(t)
	for _, path := range files {
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			source := string(content)
			opts := ast.SourceFileParseOptions{FileName: path}
			kind := scriptKindFor(path)

			want := stockparser.ParseSourceFile(opts, source, kind)
			got := etsparser.ParseSourceFile(opts, source, kind)

			if diff := compareNodes(want.AsNode(), got.AsNode()); diff != "" {
				t.Errorf("AST mismatch: %s", diff)
			}
			if diff := compareDiagnostics(want.Diagnostics(), got.Diagnostics()); diff != "" {
				t.Errorf("diagnostics mismatch: %s", diff)
			}
		})
	}
}

func compareNodes(want, got *ast.Node) string {
	if (want == nil) != (got == nil) {
		return fmt.Sprintf("node presence differs: want %v, got %v", want != nil, got != nil)
	}
	if want == nil {
		return ""
	}
	if want.Kind != got.Kind {
		return fmt.Sprintf("kind mismatch at %v: want %v, got %v", want.Loc, want.Kind, got.Kind)
	}
	if want.Loc != got.Loc {
		return fmt.Sprintf("span mismatch for %v: want %v, got %v", want.Kind, want.Loc, got.Loc)
	}
	wantChildren := childrenOf(want)
	gotChildren := childrenOf(got)
	if len(wantChildren) != len(gotChildren) {
		return fmt.Sprintf("child count mismatch for %v at %v: want %d, got %d",
			want.Kind, want.Loc, len(wantChildren), len(gotChildren))
	}
	for i := range wantChildren {
		if diff := compareNodes(wantChildren[i], gotChildren[i]); diff != "" {
			return diff
		}
	}
	return ""
}

func childrenOf(node *ast.Node) []*ast.Node {
	var children []*ast.Node
	node.ForEachChild(func(child *ast.Node) bool {
		children = append(children, child)
		return false
	})
	return children
}

func compareDiagnostics(want, got []*ast.Diagnostic) string {
	if len(want) != len(got) {
		return fmt.Sprintf("diagnostic count: want %d, got %d (first want: %v)", len(want), len(got), firstDiagnostic(want))
	}
	for i := range want {
		w, g := want[i], got[i]
		if w.Code() != g.Code() || w.Loc() != g.Loc() || w.MessageKey() != g.MessageKey() || !equalStrings(w.MessageArgs(), g.MessageArgs()) {
			return fmt.Sprintf("diagnostic %d: want TS%d %v %v at %v, got TS%d %v %v at %v",
				i, w.Code(), w.MessageKey(), w.MessageArgs(), w.Loc(), g.Code(), g.MessageKey(), g.MessageArgs(), g.Loc())
		}
	}
	return ""
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func firstDiagnostic(diags []*ast.Diagnostic) string {
	if len(diags) == 0 {
		return "<none>"
	}
	return fmt.Sprintf("TS%d %v %v at %v", diags[0].Code(), diags[0].MessageKey(), diags[0].MessageArgs(), diags[0].Loc())
}
