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
	p.inETSGenBlock++
	block := p.parseBlock(false /*ignoreMissingOpenBrace*/, (*diagnostics.Message)(nil))
	p.inETSGenBlock--
	return p.finishNode(p.factory.NewETSGenBlock(callee, block), pos)
}

// isETSRunExpression reports whether the current position starts a `run`
// expression: the identifier `run` followed, on the same line, by a token
// that can start its operand.
func (p *Parser) isETSRunExpression() bool {
	if p.token != ast.KindIdentifier || p.scanner.TokenValue() != "run" {
		return false
	}
	return p.lookAhead((*Parser).nextTokenIsETSRunOperand)
}

// nextTokenIsETSRunOperand decides whether the token after `run` begins its
// operand. Exclusions keep ordinary identifier uses of `run` intact:
//
//   - `(` `[` template: calls, element access, tagged templates
//   - `+` `-` `++` `--` `/`: binary/postfix-ambiguous (`run + 1`, `run++`)
//
// Everything else must be a token that can start an expression, which covers
// the primary case (`run getUser(1)`) as well as `run await x`, `run !x`,
// `run new X()`, and nested `run run x`.
func (p *Parser) nextTokenIsETSRunOperand() bool {
	p.nextToken()
	if p.hasPrecedingLineBreak() {
		return false
	}
	switch p.token {
	case ast.KindOpenParenToken, ast.KindOpenBracketToken, ast.KindNoSubstitutionTemplateLiteral, ast.KindTemplateHead,
		ast.KindPlusToken, ast.KindMinusToken, ast.KindPlusPlusToken, ast.KindMinusMinusToken,
		ast.KindSlashToken:
		return false
	}
	return p.isStartOfExpression()
}

// parseETSRunExpression parses `run <assignment-expression>` into an
// ETSRunExpression node. The current token must be the `run` identifier.
func (p *Parser) parseETSRunExpression() *ast.Expression {
	pos := p.nodePos()
	p.nextToken() // consume `run`
	operand := p.parseAssignmentExpressionOrHigher()
	return p.finishNode(p.factory.NewETSRunExpression(operand), pos)
}
