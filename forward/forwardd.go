package main

import (
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

const (
	MAX_SERVE_CONN_NUM = 1
)
const (
	CLIENT_INLINE = iota
	CLIENT_OFFLINE
)

type Client struct {
	Conn   *websocket.Conn
	Mutex  sync.Mutex
	Status int
}

var (
	Load                 int32
	Clients              []Client
	upgrader             = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	WS_CLOSE_ERROR_CODES = []int{000, 1001, 1002, 1003, 1004, 1005, 1006,
		1007, 1008, 1009, 1010, 1011, 1012, 1013, 1015}
	ClientsMutex = sync.Mutex{}
)

func Log(prefix string, source string, msg interface{}) {
	source_bytes := []rune(source)
	if (len(source_bytes) > 0) && source_bytes[0] >= 'a' && source_bytes[0] <= 'z' {
		source_bytes[0] = source_bytes[0] + 'A' - 'a'
	}
	source = string(source_bytes)
	log.Println("["+strings.ToUpper(prefix)+"]", source+":", msg)
}

/*
	PushHandler handle the udp to push stream to forward
*/
func PushHanler() {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 9999,
	})

	if err != nil {
		Log("err", "stream", err)
	}

	buff := make([]byte, 32768+128)
	n, _, err := conn.ReadFromUDP(buff)
	for err == nil {
		//TODO optimize to a special write thread
		//go func() {
		ClientsMutex.Lock()
		for _, c := range Clients {
			if c.Status != CLIENT_INLINE {
				Log("warning", "stream", "offline")
				continue
			}
			c.Mutex.Lock()
			c.Conn.WriteMessage(websocket.BinaryMessage, buff[0:n])
			c.Mutex.Unlock()
		}
		ClientsMutex.Unlock()
		//}()
		//Log("info", "stream", "Read "+strconv.Itoa(n)+" size stream")
		n, _, err = conn.ReadFromUDP(buff)
	}
	Log("error", "stream", err)
}

func InserClient(conn *websocket.Conn) int {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()
	for i, c := range Clients {
		if c.Status != CLIENT_INLINE {
			Clients[i].Conn = conn
			Clients[i].Status = CLIENT_INLINE
			return i
		}
	}

	Clients = append(Clients, Client{
		Conn: conn,
	})
	return len(Clients) - 1
}

func LoadHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(strconv.Itoa(int(Load))))
}

func WSHandler(w http.ResponseWriter, r *http.Request) {
	if Load >= MAX_SERVE_CONN_NUM {
		w.WriteHeader(503)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		Log("error", "resource", err)
		return
	}

	atomic.AddInt32(&Load, 1)
	defer func() {
		atomic.AddInt32(&Load, -1)
	}()

	index := InserClient(conn)
	Log("info", "wshandler", "Client "+conn.RemoteAddr().String()+" has come, index:"+strconv.Itoa(index))

	for {
		_, _, err := conn.ReadMessage()
		if websocket.IsCloseError(err, WS_CLOSE_ERROR_CODES...) {
			Log("info", "wshandler", "Client "+conn.RemoteAddr().String()+" has gone.")
			Clients[index].Status = CLIENT_OFFLINE
			return
		}
	}
}

func main() {
	go PushHanler()

	pullMux := &http.ServeMux{}
	pullMux.HandleFunc("/load", LoadHandler)
	pullMux.HandleFunc("/", WSHandler)
	pullServer := http.Server{
		Addr:    ":9998",
		Handler: pullMux,
	}
	pullServer.ListenAndServe()
}
