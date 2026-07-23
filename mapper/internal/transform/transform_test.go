package transform

import (
	"testing"

	"ets/internal/protocol"
)

func TestTransformPassThroughCoalesces(t *testing.T) {
	content := "export const answer: number = 42;\n\n// a comment between statements\n\nexport const other = \"x\";\n"
	result, err := Transform(protocol.TransformParams{
		FileName: "/project/src/answer.ets",
		Content:  content,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != content {
		t.Fatalf("expected verbatim text, got %q", result.Text)
	}
	if result.ScriptKind != scriptKindTS {
		t.Fatalf("expected ScriptKind TS (%d), got %d", scriptKindTS, result.ScriptKind)
	}
	// Per-statement verbatim segments coalesce into one whole-file mapping.
	want := protocol.SpanMapping{0, int32(len(content)), 0, int32(len(content)), spanMapKindVerbatim}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected one coalesced mapping, got %v", result.Mappings)
	}
	for i := range want {
		if result.Mappings[0][i] != want[i] {
			t.Fatalf("mapping mismatch: got %v want %v", result.Mappings[0], want)
		}
	}
}

func TestTransformParsesRealTypeScript(t *testing.T) {
	content := "const f = <T,>(x: T): T => x;\nconst re = /a+/gi;\nconst s = `tpl ${re} ${1 << 2}`;\n"
	result, err := Transform(protocol.TransformParams{FileName: "/a.ets", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != content {
		t.Fatalf("expected verbatim text, got %q", result.Text)
	}
}

func TestEmitterSegments(t *testing.T) {
	src := "abcdef"
	e := newEmitter(src)
	e.verbatim(0, 2)      // "ab" -> gen [0,2) orig [0,2)
	e.atom(2, 4, "XY")    // "XY" replaces "cd": gen [2,4) orig [2,4) atom
	e.synthesize("__")    // synthesized gap: gen [4,6)
	e.verbatim(4, 6)      // "ef" -> gen [6,8) orig [4,6)
	text, mappings := e.finish()

	if text != "abXY__ef" {
		t.Fatalf("unexpected text %q", text)
	}
	want := []protocol.SpanMapping{
		{0, 2, 0, 2, spanMapKindVerbatim},
		{2, 2, 2, 2, spanMapKindAtom},
		{6, 2, 4, 2, spanMapKindVerbatim},
	}
	if len(mappings) != len(want) {
		t.Fatalf("got %v, want %v", mappings, want)
	}
	for i := range want {
		for j := range want[i] {
			if mappings[i][j] != want[i][j] {
				t.Fatalf("mapping %d: got %v, want %v", i, mappings[i], want[i])
			}
		}
	}
}

func TestEmitterCoalescesAdjacentVerbatim(t *testing.T) {
	src := "abcdef"
	e := newEmitter(src)
	e.verbatim(0, 3)
	e.verbatim(3, 6)
	_, mappings := e.finish()
	if len(mappings) != 1 {
		t.Fatalf("expected coalesced mapping, got %v", mappings)
	}
	if mappings[0][1] != 6 || mappings[0][3] != 6 {
		t.Fatalf("unexpected coalesced lengths: %v", mappings[0])
	}
}
