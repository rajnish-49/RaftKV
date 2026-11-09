package protocol

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestEncodedResponseFrameRoundTrip(t *testing.T) {
	found, missing := true, false
	responses := []Response{
		{OK: true},
		{OK: true, Found: &found, Value: "<hello>\nworld"},
		{OK: true, Found: &found},
		{OK: true, Found: &missing},
		{Error: "value too large"},
	}
	var wire bytes.Buffer // Holds the framed bytes in memory.
	for _, response := range responses {
		payload, err := EncodeResponse(response)
		if err != nil {
			t.Fatal(err)
		}
		if err := WriteFrame(&wire, payload); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range responses {
		payload, err := ReadFrame(&wire)
		if err != nil {
			t.Fatal(err)
		}
		var got Response
		if err := json.Unmarshal(payload, &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("response after framing = %#v, want %#v", got, want)
		}
	}
}
