package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/tracker", TrackerHandler)
	http.ListenAndServe(":8080", nil)
}
