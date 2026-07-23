package etsparser_test

// Corpus smoke test: the ETS parser must parse real-world TypeScript without
// panicking or hanging. The corpus lives in the typescript-go checkout (a
// data-only dependency); the test skips when it is absent. TSGO_REPO
// overrides the location.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ets/etsparser"
	"ets/internal/ast"
	"ets/internal/core"
)

func TestCorpusSmoke(t *testing.T) {
	root := os.Getenv("TSGO_REPO")
	if root == "" {
		root = filepath.Join("..", "..", "typescript-go")
	}
	corpus := filepath.Join(root, "testdata", "tests", "cases")
	if _, err := os.Stat(corpus); err != nil {
		t.Skipf("typescript-go checkout not found at %s", corpus)
	}
	corpus, err := filepath.Abs(corpus)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	err = filepath.WalkDir(corpus, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		var kind core.ScriptKind
		switch strings.ToLower(filepath.Ext(path)) {
		case ".ts", ".mts", ".cts":
			kind = core.ScriptKindTS
		case ".tsx":
			kind = core.ScriptKindTSX
		case ".js", ".jsx", ".mjs", ".cjs":
			kind = core.ScriptKindJSX
		default:
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file := etsparser.ParseSourceFile(ast.SourceFileParseOptions{FileName: path}, string(content), kind)
		if file.Statements == nil && len(content) > 0 {
			t.Errorf("%s: parsed to nil statements", path)
		}
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("parsed %d corpus files", count)
}
