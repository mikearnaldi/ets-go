package transform_test

import (
	"testing"

	"github.com/microsoft/typescript-go/etsmapper/internal/protocol"
	"github.com/microsoft/typescript-go/etsmapper/internal/transform"
)

func TestTransformPassThrough(t *testing.T) {
	content := "export const answer: number = 42;\n"
	result, err := transform.Transform(protocol.TransformParams{
		FileName: "/project/src/answer.ets",
		Content:  content,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != content {
		t.Fatalf("expected verbatim text, got %q", result.Text)
	}
	if result.ScriptKind != 3 {
		t.Fatalf("expected ScriptKind TS (3), got %d", result.ScriptKind)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected one mapping, got %v", result.Mappings)
	}
	want := protocol.SpanMapping{0, int32(len(content)), 0, int32(len(content)), 0}
	got := result.Mappings[0]
	if len(got) != len(want) {
		t.Fatalf("expected 5-element tuple, got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mapping mismatch at %d: got %v want %v", i, got, want)
		}
	}
}

func TestTransformParsesRealTypeScript(t *testing.T) {
	content := "const f = <T,>(x: T): T => x;\nconst re = /a+/gi;\nconst s = `tpl ${re} ${1 << 2}`;\n"
	if _, err := transform.Transform(protocol.TransformParams{FileName: "/a.ets", Content: content}); err != nil {
		t.Fatal(err)
	}
}
