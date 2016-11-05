package main

import (
	"encoding/json"
	"net/http"
)

func DebugHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<b>Trackers</b><br/>"))
	data, err := json.Marshal(Trackers)
	if err != nil {
		w.WriteHeader(500)
	}
	w.Write(data)
	w.Write([]byte("<br/><b>Forwards</b><br/>"))
	data, err = json.Marshal(Forwards)
	if err != nil {
		w.WriteHeader(500)
	}
	w.Write(data)
}
