// Command ets-mapper is the ETS content mapper: it compiles .ets files to
// TypeScript for tsgo, speaking the content mapper protocol
// (microsoft/typescript-go PR #4712) over stdio.
//
// The wire format is JSON-RPC 2.0 with LSP base-protocol framing
// (Content-Length headers). The host sends all requests; this process only
// responds. Methods:
//
//	initialize -> { protocolVersion, positionEncoding, diagnosticSource }
//	transform  -> { text, scriptKind, mappings, diagnostics }
package main

import (
	"fmt"
	"os"

	"github.com/microsoft/typescript-go/etsmapper/internal/protocol"
	"github.com/microsoft/typescript-go/etsmapper/internal/transform"
)

const (
	protocolVersion  = 1
	positionEncoding = "utf-8"
	diagnosticSource = "ets"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "[ets-mapper] fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	server := protocol.NewServer(os.Stdin, os.Stdout, protocol.Handlers{
		Initialize: initialize,
		Transform:  transform.Transform,
	})
	return server.Run()
}

func initialize(params protocol.InitializeParams) (protocol.InitializeResult, error) {
	if params.ProtocolVersion != protocolVersion {
		return protocol.InitializeResult{}, fmt.Errorf("unsupported protocol version: %d", params.ProtocolVersion)
	}
	if !slicesContains(params.PositionEncodings, positionEncoding) {
		return protocol.InitializeResult{}, fmt.Errorf("host does not support %s position encoding", positionEncoding)
	}
	return protocol.InitializeResult{
		ProtocolVersion:  protocolVersion,
		PositionEncoding: positionEncoding,
		DiagnosticSource: diagnosticSource,
	}, nil
}

func slicesContains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
