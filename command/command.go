package command

import (
	"raftkv/protocol"
	"raftkv/storage"
)

func Execute(store *storage.Store, req protocol.Request) protocol.Response {
	switch req.Op {
	case protocol.OpSet:
		// safety check inside the SET case. make sure value can be read before it can be stored 
		found := true
		if _, err := protocol.EncodeResponse(protocol.Response{
			OK: true, Found: &found, Value: req.Value,
		}); err != nil {
			return protocol.Response{OK: false, Error: err.Error()}
		}
		store.Set(req.Key, req.Value)
		return protocol.Response{OK: true}

	case protocol.OpGet:
		value, found := store.Get(req.Key)
		return protocol.Response{OK: true, Found: &found, Value: value}

	case protocol.OpDelete:
		found := store.Delete(req.Key)
		return protocol.Response{OK: true, Found: &found}

	default:
		return protocol.Response{OK: false, Error: "unknown operation"}
	}
}
