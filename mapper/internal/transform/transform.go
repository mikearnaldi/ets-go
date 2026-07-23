// Package transform compiles ETS source text into TypeScript text plus a
// span map between the two, using the vendored ETS-aware scanner and parser.
//
// The pipeline:
//
//  1. Parse the .ets source with etsparser (typescript-go's parser, extended
//     with ETS productions).
//  2. Collect the ETS construct nodes and walk the source in order, emitting
//     output as segments: verbatim slices of the original source, synthesized
//     scaffolding, and atom/alias replacements. Positions are byte offsets
//     end-to-end (the negotiated position encoding is utf-8), so AST spans
//     map directly.
//  3. The emitter assembles the final text and the span map tuples from the
//     segments; gaps between mappings are synthesized content by definition.
package transform

import (
	"strings"

	"ets/etsparser"
	"ets/etsscanner"
	"ets/internal/protocol"
	"ets/internal/ast"
	"ets/internal/core"
)

const (
	scriptKindTS = 3

	spanMapKindVerbatim = 0
	spanMapKindAtom     = 1
	spanMapKindAlias    = 2
)

// Transform compiles one .ets file.
func Transform(params protocol.TransformParams) (protocol.TransformResult, error) {
	file := etsparser.ParseSourceFile(ast.SourceFileParseOptions{
		FileName: params.FileName,
	}, params.Content, core.ScriptKindTS)

	e := newEmitter(params.Content)
	f := &fileEmitter{e: e, src: params.Content, ets: collectETSNodes(file)}
	f.emitRange(0, len(params.Content))
	text, mappings := e.finish()

	return protocol.TransformResult{
		Text:       text,
		ScriptKind: scriptKindTS,
		Mappings:   mappings,
	}, nil
}

// collectETSNodes returns every ETS construct node in the file, in source
// order (a parent always precedes its nested children).
func collectETSNodes(file *ast.SourceFile) []*ast.Node {
	var nodes []*ast.Node
	var visit func(node *ast.Node) bool
	visit = func(node *ast.Node) bool {
		if node == nil {
			return false
		}
		if isETSNode(node) {
			nodes = append(nodes, node)
		}
		node.ForEachChild(visit)
		return false
	}
	file.AsNode().ForEachChild(visit)
	return nodes
}

func isETSNode(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindETSGenBlock:
		return true
	}
	return false
}

// fileEmitter emits the source in order, transforming ETS nodes as it
// encounters them.
type fileEmitter struct {
	e   *emitter
	src string
	ets []*ast.Node
	i   int
}

// emitRange emits src[start:end), transforming any ETS nodes inside.
func (f *fileEmitter) emitRange(start, end int) {
	cursor := start
	for f.i < len(f.ets) {
		node := f.ets[f.i]
		if node.Pos() < cursor {
			// Already consumed by an enclosing ETS node's handler.
			f.i++
			continue
		}
		if node.Pos() >= end {
			break
		}
		f.e.verbatim(cursor, node.Pos())
		f.i++
		cursor = f.emitETS(node)
	}
	f.e.verbatim(cursor, end)
}

func (f *fileEmitter) emitETS(node *ast.Node) int {
	switch node.Kind {
	case ast.KindETSGenBlock:
		return f.emitGenBlock(node)
	default:
		f.e.verbatim(node.Pos(), node.End())
		return node.End()
	}
}

// emitGenBlock rewrites `<callee> { body }` to `<callee>(function* () { body })`.
func (f *fileEmitter) emitGenBlock(node *ast.Node) int {
	gen := node.AsETSGenBlock()
	callee := gen.Expression
	block := gen.Block

	f.e.verbatim(node.Pos(), callee.End())
	f.e.synthesize("(function* () ")
	openBrace := etsscanner.SkipTrivia(f.src, block.Pos())
	f.e.verbatim(openBrace, openBrace+1)
	closeBrace := block.End() - 1
	f.emitRange(openBrace+1, closeBrace)
	f.e.verbatim(closeBrace, closeBrace+1)
	f.e.synthesize(")")
	return node.End()
}

// emitter accumulates output text and span mappings segment by segment.
type emitter struct {
	src       string
	out       strings.Builder
	mappings  []protocol.SpanMapping
	genOffset int
}

func newEmitter(src string) *emitter {
	return &emitter{src: src}
}

// verbatim emits src[start:end] unchanged, mapped one-to-one.
func (e *emitter) verbatim(start, end int) {
	if start >= end {
		return
	}
	e.out.WriteString(e.src[start:end])
	e.appendMapping(spanMapKindVerbatim, int32(e.genOffset), int32(end-start), int32(start), int32(end-start))
	e.genOffset += end - start
}

// atom emits text as a replacement for the original span src[start:end].
func (e *emitter) atom(start, end int, text string) {
	if text == "" {
		return
	}
	e.out.WriteString(text)
	e.appendMapping(spanMapKindAtom, int32(e.genOffset), int32(len(text)), int32(start), int32(end-start))
	e.genOffset += len(text)
}

// alias is like atom, but diagnostics display the original text of the span.
func (e *emitter) alias(start, end int, text string) {
	if text == "" {
		return
	}
	e.out.WriteString(text)
	e.appendMapping(spanMapKindAlias, int32(e.genOffset), int32(len(text)), int32(start), int32(end-start))
	e.genOffset += len(text)
}

// synthesize emits text with no corresponding location in the original
// source (a gap in the span map).
func (e *emitter) synthesize(text string) {
	if text == "" {
		return
	}
	e.out.WriteString(text)
	e.genOffset += len(text)
}

func (e *emitter) appendMapping(kind int32, genStart, genLength, origStart, origLength int32) {
	// Coalesce adjacent verbatim segments that are contiguous on both sides.
	if kind == spanMapKindVerbatim && len(e.mappings) > 0 {
		last := e.mappings[len(e.mappings)-1]
		if last[4] == spanMapKindVerbatim &&
			last[0]+last[1] == genStart &&
			last[2]+last[3] == origStart {
			last[1] += genLength
			last[3] += origLength
			e.mappings[len(e.mappings)-1] = last
			return
		}
	}
	e.mappings = append(e.mappings, protocol.NewSpanMapping(genStart, genLength, origStart, origLength, kind))
}

func (e *emitter) finish() (string, []protocol.SpanMapping) {
	return e.out.String(), e.mappings
}
