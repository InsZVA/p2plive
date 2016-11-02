package main

import (
	"net/http"
)

type Pusher struct {
	Clients []string
	Server  string
}

var ResourcePusher = make(map[int]Pusher)

func main() {
	http.HandleFunc("/get")
}
