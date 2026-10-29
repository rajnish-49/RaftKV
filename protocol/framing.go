package protocol

import (
	"encoding/binary"
	"errors"
	"io"
)

// Max 1MB.
const maxFramesize = 1 << 20

var ErrFrameTooLarge = errors.New("frame exceeds maximum payload size")

func ReadFrame(r io.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(header[:])
	if length > maxFramesize {
		return nil, ErrFrameTooLarge
	}

	payload := make([]byte, int(length))
	if _, err := io.ReadFull(r, payload); err != nil { // read until payload is completely full
		if err == io.EOF {
			err = io.ErrUnexpectedEOF // io.ErrUnexpectedEOF means the stream ended inside a frame.
		}
		return nil, err
	}
	return payload, nil
}


func WriteFrame(w io.Writer, payload []byte) error {
	if len(payload) > maxFramesize {
		return ErrFrameTooLarge
	}

	frame := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(payload)))
	copy(frame[4:], payload)

	_, err := w.Write(frame)
	return err
}
