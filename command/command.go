package command

import (
	"raftkv/protocol"
	"raftkv/storage"
)

func Execute(store *storage.Store, req protocol.Request) protocol.Response {
	switch req.Op {
	case protocol.OpSet:
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
