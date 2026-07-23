package protocol_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/microsoft/typescript-go/etsmapper/internal/protocol"
)

func frame(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return append([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))), data...)
}

func readMessage(t *testing.T, r *bytes.Buffer) map[string]any {
	t.Helper()
	var header string
	for {
		b, err := r.ReadByte()
		if err != nil {
			t.Fatal(err)
		}
		header += string(b)
		if len(header) >= 4 && header[len(header)-4:] == "\r\n\r\n" {
			break
		}
	}
	var length int
	if _, err := fmt.Sscanf(header, "Content-Length: %d\r\n\r\n", &length); err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		t.Fatal(err)
	}
	var msg map[string]any
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatal(err)
	}
	return msg
}

func TestInitializeAndTransformRoundtrip(t *testing.T) {
	var in bytes.Buffer
	var out bytes.Buffer

	server := protocol.NewServer(&in, &out, protocol.Handlers{
		Initialize: func(params protocol.InitializeParams) (protocol.InitializeResult, error) {
			return protocol.InitializeResult{
				ProtocolVersion:  1,
				PositionEncoding: "utf-8",
				DiagnosticSource: "ets",
			}, nil
		},
		Transform: func(params protocol.TransformParams) (protocol.TransformResult, error) {
			return protocol.TransformResult{
				Text:       params.Content,
				ScriptKind: 3,
				Mappings: []protocol.SpanMapping{
					protocol.NewSpanMapping(0, int32(len(params.Content)), 0, int32(len(params.Content)), 0),
				},
			}, nil
		},
	})

	in.Write(frame(t, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"protocolVersion": 1, "positionEncodings": []string{"utf-8", "utf-16"}},
	}))
	in.Write(frame(t, map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "transform",
		"params": map[string]any{"fileName": "/a.ets", "content": "let x = 1", "compilerOptions": map[string]any{}},
	}))

	if err := server.Run(); err != nil {
		t.Fatal(err)
	}

	init := readMessage(t, &out)
	if init["id"].(float64) != 1 {
		t.Fatalf("unexpected initialize id: %v", init["id"])
	}
	result := init["result"].(map[string]any)
	if result["diagnosticSource"] != "ets" || result["positionEncoding"] != "utf-8" {
		t.Fatalf("unexpected initialize result: %v", result)
	}

	tr := readMessage(t, &out)
	tres := tr["result"].(map[string]any)
	if tres["text"] != "let x = 1" {
		t.Fatalf("unexpected transform text: %v", tres["text"])
	}
	mappings := tres["mappings"].([]any)
	if len(mappings) != 1 || len(mappings[0].([]any)) != 5 {
		t.Fatalf("unexpected mappings: %v", mappings)
	}
}
