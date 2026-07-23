// Package protocol implements the content mapper wire protocol: JSON-RPC 2.0
// messages with LSP base-protocol framing, reusing typescript-go's jsonrpc
// package for transport.
package protocol

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/microsoft/typescript-go/internal/jsonrpc"
)

const (
	MethodInitialize = "initialize"
	MethodTransform  = "transform"
)

type InitializeParams struct {
	ProtocolVersion   int      `json:"protocolVersion"`
	Locale            string   `json:"locale,omitempty"`
	PositionEncodings []string `json:"positionEncodings"`
}

type InitializeResult struct {
	ProtocolVersion  int    `json:"protocolVersion"`
	PositionEncoding string `json:"positionEncoding"`
	DiagnosticSource string `json:"diagnosticSource"`
}

type TransformParams struct {
	FileName        string          `json:"fileName"`
	Content         string          `json:"content"`
	CompilerOptions json.RawMessage `json:"compilerOptions"`
}

type Diagnostic struct {
	MessageText string `json:"messageText"`
	Start       int    `json:"start"`
	Length      int    `json:"length"`
	Code        int32  `json:"code,omitempty"`
}

// SpanMapping is one span map tuple:
// [generatedStart, generatedLength, originalStart, originalLength, kind, purpose?]
// kind: 0 = Verbatim, 1 = Atom, 2 = Alias. purpose omitted = All.
type SpanMapping []int32

// NewSpanMapping builds a five-element span tuple (purpose defaults to All).
func NewSpanMapping(generatedStart, generatedLength, originalStart, originalLength, kind int32) SpanMapping {
	return SpanMapping{generatedStart, generatedLength, originalStart, originalLength, kind}
}

type TransformResult struct {
	Text        string        `json:"text"`
	ScriptKind  int           `json:"scriptKind,omitempty"`
	Mappings    []SpanMapping `json:"mappings,omitempty"`
	Diagnostics []Diagnostic  `json:"diagnostics,omitempty"`
}

type Handlers struct {
	Initialize func(InitializeParams) (InitializeResult, error)
	Transform  func(TransformParams) (TransformResult, error)
}

type Server struct {
	reader   *jsonrpc.Reader
	writer   *jsonrpc.Writer
	handlers Handlers
}

func NewServer(r io.Reader, w io.Writer, handlers Handlers) *Server {
	return &Server{
		reader: jsonrpc.NewReader(r),
		writer: jsonrpc.NewWriter(w),
		handlers: handlers,
	}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) Run() error {
	for {
		data, err := s.reader.Read()
		if err != nil {
			// io.EOF means the host closed the pipe; shut down cleanly.
			if err == io.EOF {
				return nil
			}
			return err
		}
		var req request
		if err := json.Unmarshal(data, &req); err != nil {
			s.send(response{JSONRPC: "2.0", ID: nil, Error: &responseError{Code: -32700, Message: "parse error"}})
			continue
		}
		if len(req.ID) == 0 {
			// Notification; the protocol defines none from the host. Ignore.
			continue
		}
		s.handle(&req)
	}
}

func (s *Server) handle(req *request) {
	var result any
	var err error
	switch req.Method {
	case MethodInitialize:
		var params InitializeParams
		if e := json.Unmarshal(req.Params, &params); e != nil {
			err = e
		} else {
			result, err = s.handlers.Initialize(params)
		}
	case MethodTransform:
		var params TransformParams
		if e := json.Unmarshal(req.Params, &params); e != nil {
			err = e
		} else {
			result, err = s.handlers.Transform(params)
		}
	default:
		err = fmt.Errorf("method not found: %s", req.Method)
	}
	if err != nil {
		s.send(response{JSONRPC: "2.0", ID: req.ID, Error: &responseError{Code: -32603, Message: err.Error()}})
		return
	}
	s.send(response{JSONRPC: "2.0", ID: req.ID, Result: result})
}

func (s *Server) send(resp response) {
	data, err := json.Marshal(resp)
	if err != nil {
		data = []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"marshal error"}}`)
	}
	// Transport errors are fatal to the session; Run will return them on the next read.
	_ = s.writer.Write(data)
}
