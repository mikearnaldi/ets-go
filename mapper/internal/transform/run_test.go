package transform

import (
	"testing"

	"ets/internal/protocol"
)

func transformText(t *testing.T, content string) string {
	t.Helper()
	result, err := Transform(protocol.TransformParams{FileName: "/a.ets", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	return result.Text
}

func TestRunExpression(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "variable initializer",
			content: "Effect.gen {\n  const user = run getUser(1)\n}\n",
			want:    "Effect.gen(function* () {\n  const user = yield* getUser(1)\n})\n",
		},
		{
			name:    "expression statement",
			content: "Effect.gen {\n  run getUser(1)\n}\n",
			want:    "Effect.gen(function* () {\n  yield* getUser(1)\n})\n",
		},
		{
			name:    "return",
			content: "Effect.gen {\n  return run getUser(1)\n}\n",
			want:    "Effect.gen(function* () {\n  return yield* getUser(1)\n})\n",
		},
		{
			name:    "call argument",
			content: "Effect.gen {\n  foo(run getUser(1))\n}\n",
			want:    "Effect.gen(function* () {\n  foo(yield* getUser(1))\n})\n",
		},
		{
			name:    "binary operand",
			content: "Effect.gen {\n  const x = run a + run b\n}\n",
			want:    "Effect.gen(function* () {\n  const x = yield* a + yield* b\n})\n",
		},
		{
			name:    "nested run",
			content: "Effect.gen {\n  const x = run run a\n}\n",
			want:    "Effect.gen(function* () {\n  const x = yield* yield* a\n})\n",
		},
		{
			name:    "await operand",
			content: "Effect.gen {\n  const x = run await a\n}\n",
			want:    "Effect.gen(function* () {\n  const x = yield* await a\n})\n",
		},
		{
			name:    "nested gen block operand",
			content: "Effect.gen {\n  const x = run Effect.gen {\n    return 1\n  }\n}\n",
			want:    "Effect.gen(function* () {\n  const x = yield* Effect.gen(function* () {\n    return 1\n  })\n})\n",
		},
		{
			name:    "run inside arrow inside gen block",
			content: "Effect.gen {\n  const f = () => run a\n}\n",
			want:    "Effect.gen(function* () {\n  const f = () => yield* a\n})\n",
		},

		// Disambiguation: `run` stays an ordinary identifier.
		{
			name:    "call expression",
			content: "Effect.gen {\n  run(getUser(1))\n}\n",
			want:    "Effect.gen(function* () {\n  run(getUser(1))\n})\n",
		},
		{
			name:    "call with space",
			content: "Effect.gen {\n  run (getUser(1))\n}\n",
			want:    "Effect.gen(function* () {\n  run (getUser(1))\n})\n",
		},
		{
			name:    "member access",
			content: "Effect.gen {\n  run.x\n}\n",
			want:    "Effect.gen(function* () {\n  run.x\n})\n",
		},
		{
			name:    "element access",
			content: "Effect.gen {\n  run[0]\n}\n",
			want:    "Effect.gen(function* () {\n  run[0]\n})\n",
		},
		{
			name:    "assignment",
			content: "Effect.gen {\n  run = 1\n}\n",
			want:    "Effect.gen(function* () {\n  run = 1\n})\n",
		},
		{
			name:    "binary plus",
			content: "Effect.gen {\n  const x = run + 1\n}\n",
			want:    "Effect.gen(function* () {\n  const x = run + 1\n})\n",
		},
		{
			name:    "binary minus",
			content: "Effect.gen {\n  const x = run - 1\n}\n",
			want:    "Effect.gen(function* () {\n  const x = run - 1\n})\n",
		},
		{
			name:    "postfix increment",
			content: "Effect.gen {\n  run++\n}\n",
			want:    "Effect.gen(function* () {\n  run++\n})\n",
		},
		{
			name:    "arrow function",
			content: "Effect.gen {\n  const f = run => run + 1\n}\n",
			want:    "Effect.gen(function* () {\n  const f = run => run + 1\n})\n",
		},
		{
			name:    "standalone identifier",
			content: "Effect.gen {\n  run\n}\n",
			want:    "Effect.gen(function* () {\n  run\n})\n",
		},
		{
			name:    "line break is not run expression",
			content: "Effect.gen {\n  run\n  getUser(1)\n}\n",
			want:    "Effect.gen(function* () {\n  run\n  getUser(1)\n})\n",
		},

		// Outside a gen block, `run` is never special.
		{
			name:    "outside gen block",
			content: "const x = run getUser(1)\n",
			want:    "const x = run getUser(1)\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := transformText(t, tc.content); got != tc.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tc.want)
			}
		})
	}
}

func TestRunExpressionMapping(t *testing.T) {
	content := "Effect.gen {\n  const x = run foo\n}\n"
	result, err := Transform(protocol.TransformParams{FileName: "/a.ets", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	// The `run` keyword maps to `yield*` as an alias; everything else is verbatim.
	var found bool
	for _, m := range result.Mappings {
		if m[4] == spanMapKindAlias {
			found = true
			if m[3] != 3 || m[1] != 6 {
				t.Errorf("alias span: got gen len %d orig len %d, want 6 and 3", m[1], m[3])
			}
			if content[m[2]:m[2]+m[3]] != "run" {
				t.Errorf("alias original text: got %q, want %q", content[m[2]:m[2]+m[3]], "run")
			}
			if result.Text[m[0]:m[0]+m[1]] != "yield*" {
				t.Errorf("alias generated text: got %q, want %q", result.Text[m[0]:m[0]+m[1]], "yield*")
			}
		}
	}
	if !found {
		t.Errorf("no alias mapping found in %v", result.Mappings)
	}
}
