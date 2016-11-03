package main

import (
	"encoding/json"
	"net/http"
)

func DebugHandler(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(Trackers)
	if err != nil {
		w.WriteHeader(500)
	}
	w.Write(data)
}
