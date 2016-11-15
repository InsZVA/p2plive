package main

import (
	"net/http"
	"time"
)

func StreamHandler(w http.ResponseWriter, r *http.Request) {
	Log("info", "stream", r.Method)
	defer r.Body.Close()

	buff := make([]byte, 32768)
	n, err := r.Body.Read(buff)
	for err == nil {
		if ForwardLastUpdateTime.Add(FORWARD_UPDATE_INTERVAL*time.Second).Before(time.Now()) &&
			UpdateState == FORWARD_READY {
			go UpdateForwards()
		}

		for _, f := range Forwards {
			ForwardStream(buff, n, f)
		}

		n, err = r.Body.Read(buff)

		if UpdateState == FORWARD_UPDATED {
			ApplyUpdate()
		}
	}
	Log("error", "stream", err)
}
