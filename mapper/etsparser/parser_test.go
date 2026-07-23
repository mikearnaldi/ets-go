package etsparser_test

import (
	"strings"
	"testing"

	"ets/etsparser"
	"ets/internal/ast"
	"ets/internal/core"
	"gotest.tools/v3/assert"
)

func TestJSDocImportTypeParentChain(t *testing.T) {
	t.Parallel()
	sourceText := `test("", async function () {
  ;(/** @type {typeof import("a")} */ ({}))
})

test("", async function () {
  ;(/** @type {typeof import("a")} */ a)
})

test("", async function () {
  (/** @type {typeof import("a")} */ ({}))
  ;(/** @type {typeof import("a")} */ ({}))
})

test("", async function () {
  (/** @type {typeof import("a")} */ a)
  ;(/** @type {typeof import("a")} */ a)
})

test("", async function () {
  (/** @type {typeof import("a")} */ ({}))
  ;(/** @type {typeof import("a")} */ ({}))
})
`
	opts := ast.SourceFileParseOptions{
		FileName: "/index.js",
		Path:     "/index.js",
	}

	file := etsparser.ParseSourceFile(opts, sourceText, core.ScriptKindJS)

	for i := 1; i < len(file.ReparsedClones); i++ {
		a, b := file.ReparsedClones[i-1], file.ReparsedClones[i]
		if a.Pos() == b.Pos() && a.End() == b.End() && a.Kind == b.Kind {
			t.Errorf("duplicate ReparsedClones at [%d] and [%d]: %s pos=%d end=%d", i-1, i, a.Kind.String(), a.Pos(), a.End())
		}
	}

	for _, imp := range file.Imports() {
		reparsed := ast.GetReparsedNodeForNode(imp)
		if ast.GetSourceFileOfNode(reparsed) == nil {
			t.Errorf("reparsed import at pos=%d has broken parent chain", imp.Pos())
		}
	}
}

func TestSourceFileContainsNonASCIIInStringLiteralFastPath(t *testing.T) {
	t.Parallel()
	sourceText := `const x = "─";

namespace N {
  export const y = x;
}
`
	opts := ast.SourceFileParseOptions{
		FileName: "/index.ts",
		Path:     "/index.ts",
	}

	file := etsparser.ParseSourceFile(opts, sourceText, core.ScriptKindTS)

	assert.Assert(t, file.ContainsNonASCII)
	positionMap := file.GetPositionMap()
	assert.Assert(t, !positionMap.IsAsciiOnly())
	afterBoxDrawingCharacter := strings.Index(sourceText, "─") + len("─")
	assert.Equal(t, positionMap.UTF8ToUTF16(afterBoxDrawingCharacter), afterBoxDrawingCharacter-2)
	assert.Equal(t, positionMap.UTF8ToUTF16(len(sourceText)), len(sourceText)-2)
}
