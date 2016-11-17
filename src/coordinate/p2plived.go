package main

import (
	"log"
	"net/http"
	"strings"
)

func Log(prefix string, source string, msg interface{}) {
	source_bytes := []rune(source)
	if (len(source_bytes) > 0) && source_bytes[0] >= 'a' && source_bytes[0] <= 'z' {
		source_bytes[0] = source_bytes[0] + 'A' - 'a'
	}
	source = string(source_bytes)
	log.Println("["+strings.ToUpper(prefix)+"]", source+":", msg)
}

func main() {
	http.HandleFunc("/tracker", TrackerHandler)
	http.HandleFunc("/stream", StreamHandler)
	http.HandleFunc("/debug", DebugHandler)
	http.ListenAndServe(":8080", nil)
}
