package main

import (
	"bytes"
	"net/http"
	"time"
)

var (
	SEQ_END_CODE       = []byte{0x00, 0x00, 0x01, 0xb7}
	SEQ_START_CODE     = []byte{0x00, 0x00, 0x01, 0xb3}
	PICTURE_START_CODE = []byte{0x00, 0x00, 0x01, 0x00}
	GOP_START_CODE     = []byte{0x00, 0x00, 0x01, 0xb8}
)

func StreamHandler(w http.ResponseWriter, r *http.Request) {
	Log("info", "stream", r.Method)
	defer r.Body.Close()

	seq := make([]byte, 0)

	buff := make([]byte, 32768)
	n, err := r.Body.Read(buff)
	startPos := 0

	for err == nil {
		if ForwardLastUpdateTime.Add(FORWARD_UPDATE_INTERVAL*time.Second).Before(time.Now()) &&
			UpdateState == FORWARD_READY {
			UpdateForwards()
		}

		for startPos = bytes.Index(buff, GOP_START_CODE); startPos == -1 && err == nil; startPos = bytes.Index(buff, GOP_START_CODE) {
			seq = append(seq, buff[:n]...)
			n, err = r.Body.Read(buff)
		}
		seq = append(seq, buff[:startPos]...)
		if len(seq) != 0 {
			for _, f := range Forwards {
				ForwardStream(seq, len(seq), f)
			}
		}

		seq = make([]byte, 0)
		seq = append(seq, buff[startPos:n]...)
		n, err = r.Body.Read(buff)

		if UpdateState == FORWARD_UPDATED {
			ApplyUpdate()
		}
	}
	/*
		for err == nil {
			if ForwardLastUpdateTime.Add(FORWARD_UPDATE_INTERVAL*time.Second).Before(time.Now()) &&
				UpdateState == FORWARD_READY {
				UpdateForwards()
			}

			for _, f := range Forwards {
				ForwardStream(buff, n, f)
			}

			n, err = r.Body.Read(buff)

			if UpdateState == FORWARD_UPDATED {
				ApplyUpdate()
			}
		}*/
	Log("error", "stream", err)
}
