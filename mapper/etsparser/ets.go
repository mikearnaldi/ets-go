package etsparser

// ETS grammar extensions to the vendored typescript-go parser. This file is
// ETS-owned (not vendored); the only modification to vendored files is the
// hook in parseAssignmentExpressionOrHigherWorker, marked with an ETS comment.

import (
	"ets/internal/ast"
	"ets/internal/diagnostics"
)

// isETSTrailingBlockCallee reports whether expr can be the callee of a
// trailing block call: `<callee> { statements }`.
//
// Only member expressions qualify. A bare identifier callee (`gen { }`) is
// deliberately rejected: `ident {` in statement position is how several
// common typos look (e.g. a missing keyword before a block, as in the
// typescript-go test corpus file unicodeSpellingSuggestions.ts), and stock
// TypeScript reports those as errors. Keeping it an error preserves both
// that DX and error-recovery parity with the stock parser.
func isETSTrailingBlockCallee(expr *ast.Expression) bool {
	return expr.Kind == ast.KindPropertyAccessExpression
}

// parseETSTrailingBlock parses the `{ statements }` following a trailing
// block callee and produces an ETSGenBlock node spanning the callee and the
// block. The current token must be the opening brace.
func (p *Parser) parseETSTrailingBlock(callee *ast.Expression) *ast.Expression {
	pos := callee.Pos()
	block := p.parseBlock(false /*ignoreMissingOpenBrace*/, (*diagnostics.Message)(nil))
	return p.finishNode(p.factory.NewETSGenBlock(callee, block), pos)
}
