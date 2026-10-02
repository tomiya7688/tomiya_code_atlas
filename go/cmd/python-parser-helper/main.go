// Command python-parser-helper implements Parser Backend Contract v1 for Python.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/tomiya7688/tomiya_code_atlas/internal/commonir"
	"github.com/tomiya7688/tomiya_code_atlas/internal/pythonadapter"
)

type request struct {
	ContractVersion string `json:"contract_version"`
	RequestID       string `json:"request_id"`
	Operation       string `json:"operation"`
	Language        string `json:"language"`
	Source          string `json:"source"`
	Path            string `json:"path,omitempty"`
}
type failure struct {
	Kind      string `json:"kind"`
	Message   string `json:"message"`
	BackendID string `json:"backend_id"`
	Retryable bool   `json:"retryable"`
}
type response struct {
	ContractVersion string            `json:"contract_version"`
	RequestID       string            `json:"request_id"`
	OK              bool              `json:"ok"`
	IR              *commonir.Payload `json:"ir,omitempty"`
	Error           *failure          `json:"error,omitempty"`
}

func handle(req request) response {
	if req.ContractVersion != "1" || req.RequestID == "" || req.Operation != "parse" || req.Language != "python" {
		return response{ContractVersion: "1", RequestID: req.RequestID, OK: false, Error: &failure{Kind: "protocol_error", Message: "contract_version 1, request_id, operation=parse, and language=python are required", BackendID: "python-source-subset-adapter", Retryable: false}}
	}
	ir := pythonadapter.Parse(req.Source)
	if err := pythonadapter.Validate(ir); err != nil {
		return response{ContractVersion: "1", RequestID: req.RequestID, OK: false, Error: &failure{Kind: "failure", Message: err.Error(), BackendID: "python-source-subset-adapter", Retryable: false}}
	}
	return response{ContractVersion: "1", RequestID: req.RequestID, OK: true, IR: &ir}
}

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var req request
	if err = json.Unmarshal(data, &req); err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(response{ContractVersion: "1", OK: false, Error: &failure{Kind: "protocol_error", Message: "invalid JSON request", BackendID: "python-source-subset-adapter", Retryable: false}})
		return
	}
	_ = json.NewEncoder(os.Stdout).Encode(handle(req))
}
