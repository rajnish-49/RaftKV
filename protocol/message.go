package protocol

import (
	"encoding/json"
	"fmt"
)

const (
	OpSet    = "SET"
	OpGet    = "GET"
	OpDelete = "DELETE"
)

type Request struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Response struct {
	OK    bool   `json:"ok"` // ok means command was processed
	Found *bool  `json:"found,omitempty"`
	Value string `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

// turn payload into a reuqest that can be trusted
func DecodeRequest(payload []byte) (Request, error) {
	var req Request // json gets decoded into it
	if err := json.Unmarshal(payload, &req); err != nil {
		return Request{}, fmt.Errorf("invalid JSON: %w", err)
	}

	switch req.Op {
	case OpSet, OpGet, OpDelete:

	default:
		return Request{}, fmt.Errorf("unknown operation: %q", req.Op)
	}

	if req.Key == "" {
		return Request{}, fmt.Errorf("key must not be empty")
	}

	return req, nil
}
