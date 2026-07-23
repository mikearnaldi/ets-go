package transform

import (
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/etsmapper/internal/protocol"
)

func TestTransformGenBlock(t *testing.T) {
	content := "const program = Effect.gen {\n  const x = 1\n  return x\n}\n"
	want := "const program = Effect.gen(function* () {\n  const x = 1\n  return x\n})\n"

	result, err := Transform(protocol.TransformParams{FileName: "/a.ets", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != want {
		t.Fatalf("unexpected text:\n%q\nwant:\n%q", result.Text, want)
	}

	// Expected segments: verbatim up to and including the callee, synthesized
	// "(function* () ", verbatim "{ body }", synthesized ")", verbatim "\n".
	// The brace, body, and closing brace coalesce into one verbatim segment.
	calleeEnd := strings.Index(content, " {")
	braceOpen := strings.Index(content, "{")
	braceClose := strings.LastIndex(content, "}")
	genBraceOpen := strings.Index(want, "{")
	genBraceClose := strings.LastIndex(want, "}")

	wantMappings := []protocol.SpanMapping{
		{0, int32(calleeEnd), 0, int32(calleeEnd), spanMapKindVerbatim},
		{int32(genBraceOpen), int32(genBraceClose + 1 - genBraceOpen), int32(braceOpen), int32(braceClose + 1 - braceOpen), spanMapKindVerbatim},
		{int32(len(want) - 1), 1, int32(len(content) - 1), 1, spanMapKindVerbatim},
	}
	if len(result.Mappings) != len(wantMappings) {
		t.Fatalf("mappings: got %v, want %v", result.Mappings, wantMappings)
	}
	for i := range wantMappings {
		for j := range wantMappings[i] {
			if result.Mappings[i][j] != wantMappings[i][j] {
				t.Fatalf("mapping %d: got %v, want %v", i, result.Mappings[i], wantMappings[i])
			}
		}
	}
}

func TestTransformGenBlockNested(t *testing.T) {
	content := "const program = Effect.gen {\n  const inner = Effect.gen {\n    return 1\n  }\n}\n"
	want := "const program = Effect.gen(function* () {\n  const inner = Effect.gen(function* () {\n    return 1\n  })\n})\n"

	result, err := Transform(protocol.TransformParams{FileName: "/a.ets", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != want {
		t.Fatalf("unexpected text:\n%q\nwant:\n%q", result.Text, want)
	}
}

func TestTransformGenBlockAsExpression(t *testing.T) {
	content := "Effect.gen { return 1 }\n"
	want := "Effect.gen(function* () { return 1 })\n"

	result, err := Transform(protocol.TransformParams{FileName: "/a.ets", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != want {
		t.Fatalf("unexpected text:\n%q\nwant:\n%q", result.Text, want)
	}
}

func TestTransformNewlineBeforeBraceIsNotGenBlock(t *testing.T) {
	// A line break between the callee and the brace disables the construct
	// (restricted production); the text is erroneous TS and passes through
	// untouched for TypeScript to diagnose.
	content := "Effect.gen\n{\n}\n"
	result, err := Transform(protocol.TransformParams{FileName: "/a.ets", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != content {
		t.Fatalf("expected verbatim text, got %q", result.Text)
	}
}

func TestTransformBareIdentifierBlockIsNotGenBlock(t *testing.T) {
	// Bare identifier callees are not trailing block calls; stock TypeScript
	// reports `ident {` as an error and we preserve that.
	content := "gen {\n}\n"
	result, err := Transform(protocol.TransformParams{FileName: "/a.ets", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != content {
		t.Fatalf("expected verbatim text, got %q", result.Text)
	}
}
