"use strict";

// ETS -> TypeScript transform.
//
// This is currently a pass-through: the ETS surface syntax has not been
// designed yet, so .ets files are expected to contain valid TypeScript
// and the whole file maps verbatim. When ETS syntax (Schema, Service,
// Generators, ...) is defined, this module will emit transformed text
// plus span mappings between the generated and original content.
//
// Span mapping tuple:
//   [generatedStart, generatedLength, originalStart, originalLength, kind, purpose?]
// kind:    0 = Verbatim, 1 = Atom, 2 = Alias
// purpose: 0 = None, 1 = Semantic, 2 = Navigation, 3 = All (default when omitted)
// Positions use the position encoding selected at initialize (utf-16).

const ScriptKind = { JS: 1, JSX: 2, TS: 3, TSX: 4, JSON: 6 };
const SpanMapKind = { Verbatim: 0, Atom: 1, Alias: 2 };

/**
 * @param {{ fileName: string, content: string, compilerOptions: Record<string, unknown> }} params
 */
function transform({ fileName, content, compilerOptions }) {
  void fileName;
  void compilerOptions;
  return {
    text: content,
    scriptKind: ScriptKind.TS,
    mappings: [
      [0, content.length, 0, content.length, SpanMapKind.Verbatim],
    ],
  };
}

module.exports = { transform, ScriptKind, SpanMapKind };
