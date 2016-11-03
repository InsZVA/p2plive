package main

import (
	"net/http"
	"strings"
	"sync"

	"sync/atomic"

	"github.com/gorilla/websocket"
)

const (
	MAX_LOAD = 1000

	// The number the forward server can service best
	// If clients number is less than it,
	// client will pull from forward server directly
	FORWARD_BEST_SERVICE_NUM = 1

	USERAGENT_MAX_PUSHNUM = 2

	USERAGENT_READY = iota
	USERAGENT_AWAY
)

type UserAgent struct {
	IPAddress string
	Status    int
	// 1 means direct from forward server, -1 mean none
	ForwardTimes int
	// Number of clients this user agent pull from
	PullNum int
	// Number of clients this user agent push to
	PushNum int
	// TODO statistics the delay of this user agent
	Delay []int
	Conn  *websocket.Conn
}

var (
	Load                 int64
	upgrader             = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	WS_CLOSE_ERROR_CODES = []int{000, 1001, 1002, 1003, 1004, 1005, 1006,
		1007, 1008, 1009, 1010, 1011, 1012, 1013, 1015}

	Clients      = make(map[string]UserAgent)
	ClientsMutex = sync.Mutex{}
)

func NewClient(ipaddress string, conn *websocket.Conn) *UserAgent {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()
	userAgent := UserAgent{
		IPAddress:    ipaddress,
		ForwardTimes: 0,
		Delay:        []int{},
		Conn:         conn,
	}
	Clients[ipaddress] = userAgent
	return
}

func RemoveClient(ipaddress string) {
	ClientsMutex.Lock()
	if c, ok := Clients[ipaddress]; ok {
		c.Status = USERAGENT_AWAY
		delete(Clients, ipaddress)
	}
	ClientsMutex.Unlock()
}

/*
	PeekSourceClient peek a good client to push P2P stream to another
	Returns "" when it should be a forward server better
*/
func PeekSourceClient() string {
	ClientsMutex.Lock()
	if len(Clients) < FORWARD_BEST_SERVICE_NUM*ForwardsAvaliable {
		return nil
	}
	count := 0
	best := ""
	for ipaddress, client := range Clients {
		if client.PushNum >= USERAGENT_MAX_PUSHNUM {
			continue
		}
		if best == "" || client.ForwardTimes < Clients[best].ForwardTimes {
			best = ipaddress
		}
		// To save time, search 50 clients too more
		if count > 50 {
			break
		}
		count++
	}
	return best
}

func PeekSourceServer() string {
	ForwardsMutex.Lock()
	defer ForwardsMutex.Unlock()
	var best *ForwardServer
	for i, forward := range Forwards {
		if best == nil || forward.Load < best.Load {
			best = &Forwards[i]
		}
	}
	return best.PullStreamAddress
}

func MakePeerConnection(pullerIpAddress string, pusherIpAddress string) {
	// TODO Implement this
}

func ResourceHandler(w http.ResponseWriter, r *http.Request) {
	if ForwardsAvaliable == 0 {
		w.WriteHeader(503)
		return
	}

	atomic.AddInt64(&Load, 1)
	defer func() {
		atomic.AddInt64(&Load, -1)
	}()

	if Load >= MAX_LOAD {
		w.WriteHeader(503)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		Log("error", "resource", err)
		return
	}
	ipaddress := strings.Split(r.RemoteAddr, ":")[0]
	NewClient(ipaddress, conn)
	defer conn.Close(ipaddress)
	defer recover()

	for {
		msg := make(map[string]interface{})
		err := conn.ReadJSON(&msg)
		if websocket.IsCloseError(err, WS_CLOSE_ERROR_CODES) {
			RemoveClient(ipaddress)
			conn.Close()
			return
		}
		if err != nil {
			Log("error", "resource", err)
			continue
		}

		method, ok := msg["method"].(string)
		if !ok {
			Log("error", "resource", "method missing")
			continue
		}

		switch method {
		case "getSource":
			if Clients[ipaddress].PullNum < 2 {
				pusherIpAddress := PeekSourceClient()
				if pusherAddress == "" {
					forwardAddress := PeekSourceServer()
					Clients[ipaddress].ForwardTimes = 1
					Clients[ipaddress].PullNum = 1
				} else {
					MakePeerConnection(ipaddress, pusherAddress)
				}
			}
		case "update":
			if pullNum, ok := msg["pullNum"]; ok {
				Clients[ipaddress].PullNum = pullNum
			}
			if pushNum, ok := msg["pushNum"]; ok {
				Clients[ipaddress].PushNum = pushNum
			}
		}
	}
}
