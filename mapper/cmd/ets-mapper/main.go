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

	"ets/internal/protocol"
	"ets/internal/transform"
)

const (
	protocolVersion  = 1
	positionEncoding = "utf-8"
	diagnosticSource = "ets"
)

func main() {
	// Hidden debug mode: `ets-mapper debug <file>` prints the transformed
	// text and span mappings for a single .ets file.
	if len(os.Args) == 3 && os.Args[1] == "debug" {
		if err := debug(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "[ets-mapper] debug: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "[ets-mapper] fatal: %v\n", err)
		os.Exit(1)
	}
}

func debug(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	result, err := transform.Transform(protocol.TransformParams{
		FileName: path,
		Content:  string(content),
	})
	if err != nil {
		return err
	}
	fmt.Println("=== text ===")
	fmt.Print(result.Text)
	fmt.Println("=== mappings ===")
	for _, m := range result.Mappings {
		fmt.Printf("gen[%d:%d] -> orig[%d:%d] kind=%d\n", m[0], m[0]+m[1], m[2], m[2]+m[3], m[4])
	}
	return nil
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
