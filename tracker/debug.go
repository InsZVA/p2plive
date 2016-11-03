package main

import (
	"encoding/json"
	"net/http"
)

func DebugHandler(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(Config)
	if err != nil {
		w.WriteHeader(500)
	}
	w.Write(data)
	w.Write([]byte("<br/>"))
	data, err = json.Marshal(Forwards)
	if err != nil {
		w.WriteHeader(500)
	}
	w.Write(data)
}
