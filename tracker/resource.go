package main

import (
	"net/http"
)

type UserAgent struct {
	IPAddress    string
	Status       int
	ForwardTimes int
	PullNum      int
	PushNum      int
}

var (
	Load int
)

func ResourceHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":

	}
}
