package protocol

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestEncodeResponse(t *testing.T) {
	found, missing := true, false
	tests := []struct {
		name     string
		response Response
		want     string
	}{
		{name: "set success", response: Response{OK: true}, want: `{"ok":true}`},
		{name: "found value", response: Response{OK: true, Found: &found, Value: "Alice"}, want: `{"ok":true,"found":true,"value":"Alice"}`},
		{name: "empty value", response: Response{OK: true, Found: &found}, want: `{"ok":true,"found":true}`},
		{name: "missing key", response: Response{OK: true, Found: &missing}, want: `{"ok":true,"found":false}`},
		{name: "escaped value", response: Response{OK: true, Found: &found, Value: "<\n"}, want: `{"ok":true,"found":true,"value":"\u003c\n"}`},
		{name: "error", response: Response{Error: "unknown operation"}, want: `{"ok":false,"error":"unknown operation"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := EncodeResponse(tt.response)
			if err != nil {
				t.Fatal(err)
			}
			if string(payload) != tt.want {
				t.Errorf("encoded response = %s, want %s", payload, tt.want)
			}
		})
	}
}

func TestEncodeResponseSizeLimit(t *testing.T) {
	found := true
	const overhead = len(`{"ok":true,"found":true,"value":""}`)
	for _, extra := range []int{0, 1} {
		t.Run(fmt.Sprintf("extra=%d", extra), func(t *testing.T) {
			response := Response{
				OK: true, Found: &found,
				Value: strings.Repeat("a", maxFramesize-overhead+extra),
			}
			payload, err := EncodeResponse(response)
			if extra == 1 {
				if !errors.Is(err, ErrFrameTooLarge) || payload != nil {
					t.Fatalf("oversized response: got %d bytes, %v; want nil, ErrFrameTooLarge", len(payload), err)
				}
				return
			}
			if err != nil || len(payload) != maxFramesize {
				t.Fatalf("maximum response: got %d bytes, %v; want %d bytes, nil", len(payload), err, maxFramesize)
			}
		})
	}
}

func TestDecodeRequest(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    Request
		wantErr string
	}{
		{
			name:    "set",
			payload: `{"op":"SET","key":"name","value":"Alice"}`,
			want:    Request{Op: "SET", Key: "name", Value: "Alice"},
		},
		{
			name:    "get",
			payload: `{"op":"GET","key":"name"}`,
			want:    Request{Op: "GET", Key: "name"},
		},
		{
			name:    "delete",
			payload: `{"op":"DELETE","key":"name"}`,
			want:    Request{Op: "DELETE", Key: "name"},
		},
		{
			name:    "empty value",
			payload: `{"op":"SET","key":"name","value":""}`,
			want:    Request{Op: "SET", Key: "name"},
		},
		{
			name:    "spaces and newline in value",
			payload: `{"op":"SET","key":"message","value":"hello world\nnext line"}`,
			want:    Request{Op: "SET", Key: "message", Value: "hello world\nnext line"},
		},
		{name: "malformed JSON", payload: `{"op":`, wantErr: "invalid JSON"},
		{name: "wrong field type", payload: `{"op":"GET","key":123}`, wantErr: "invalid JSON"},
		{name: "multiple JSON messages", payload: `{"op":"GET","key":"a"}{"op":"GET","key":"b"}`, wantErr: "invalid JSON"},
		{name: "unknown operation", payload: `{"op":"INCR","key":"count"}`, wantErr: "unknown operation"},
		{name: "lowercase operation", payload: `{"op":"get","key":"name"}`, wantErr: "unknown operation"},
		{name: "missing operation", payload: `{"key":"name"}`, wantErr: "unknown operation"},
		{name: "null request", payload: `null`, wantErr: "unknown operation"},
		{name: "missing key", payload: `{"op":"GET"}`, wantErr: "key must not be empty"},
		{name: "empty key", payload: `{"op":"GET","key":""}`, wantErr: "key must not be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeRequest([]byte(tt.payload))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("DecodeRequest error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("DecodeRequest = %#v, want %#v", got, tt.want)
			}
		})
	}
}
