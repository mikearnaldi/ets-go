// Package transform compiles ETS source text into TypeScript text plus a
// span map between the two, using the vendored ETS-aware scanner and parser.
//
// The pipeline currently performs a full parse (exercising the vendored
// parser end-to-end) and emits the input verbatim with a whole-file mapping.
// ETS constructs will be rewritten here as the language is designed.
package transform

import (
	"github.com/microsoft/typescript-go/etsmapper/internal/protocol"
	"github.com/microsoft/typescript-go/etsmapper/etsparser"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
)

const (
	scriptKindTS = 3

	spanMapKindVerbatim = 0
	spanMapKindAtom     = 1
	spanMapKindAlias    = 2
)

// Transform compiles one .ets file. Positions in the result use the
// position encoding negotiated at initialize (utf-8, i.e. byte offsets).
func Transform(params protocol.TransformParams) (protocol.TransformResult, error) {
	// Parse with the ETS-aware parser. Parse errors are intentionally not
	// reported here: TypeScript will diagnose the transformed text itself,
	// and ETS-level syntax errors will become mapper-authored diagnostics
	// once the ETS grammar lands.
	_ = etsparser.ParseSourceFile(ast.SourceFileParseOptions{
		FileName: params.FileName,
	}, params.Content, core.ScriptKindTS)

	text := params.Content
	return protocol.TransformResult{
		Text:       text,
		ScriptKind: scriptKindTS,
		Mappings: []protocol.SpanMapping{
			protocol.NewSpanMapping(0, int32(len(text)), 0, int32(len(params.Content)), spanMapKindVerbatim),
		},
	}, nil
}
