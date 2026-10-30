package protocol

import (
	"strings"
	"testing"
)

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
