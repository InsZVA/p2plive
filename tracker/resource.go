package main

import (
	"net/http"
	"strconv"
	_ "strings"
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

	USERAGENT_MAX_PUSHNUM = 3

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

	ConnMutex sync.Mutex
	Conn      *websocket.Conn
}

var (
	Load                 int64
	upgrader             = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	WS_CLOSE_ERROR_CODES = []int{000, 1001, 1002, 1003, 1004, 1005, 1006,
		1007, 1008, 1009, 1010, 1011, 1012, 1013, 1015}

	Clients              = make(map[string]UserAgent)
	ClientsMutex         = sync.Mutex{}
	ClientsNumDirectPull int32
)

func NewClient(ipaddress string, conn *websocket.Conn) {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()
	userAgent := UserAgent{
		IPAddress:    ipaddress,
		ForwardTimes: 0,
		Delay:        []int{},
		Conn:         conn,
		PullNum:      0,
		PushNum:      0,
	}
	Clients[ipaddress] = userAgent
	Log("info", "resource", "client:"+ipaddress+" has connected.")
}

func RemoveClient(ipaddress string) {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()
	if c, ok := Clients[ipaddress]; ok {
		c.Status = USERAGENT_AWAY
		if c.ForwardTimes == 1 {
			atomic.AddInt32(&ClientsNumDirectPull, -1)
		}
		delete(Clients, ipaddress)
	}
	Log("info", "resource", "client:"+ipaddress+" has been removed.")
}

/*
	PeekSourceClient peek a good client to push P2P stream to another
	Returns "" when it should be a forward server better
*/
func PeekSourceClient() string {
	ClientsMutex.Lock()
	defer ClientsMutex.Unlock()
	if int(atomic.LoadInt32(&ClientsNumDirectPull)) < FORWARD_BEST_SERVICE_NUM*ForwardsAvaliable {
		Log("info", "sourcepeeker", "DirectClient:"+strconv.Itoa(int(atomic.LoadInt32(&ClientsNumDirectPull)))+
			" ForwardAvaildable:"+strconv.Itoa(ForwardsAvaliable)+" so make a direct pull")
		return ""
	}
	count := 0
	best := ""
	for ipaddress, client := range Clients {
		if client.PushNum >= USERAGENT_MAX_PUSHNUM {
			continue
		}
		if client.PullNum == 0 {
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
	ClientsMutex.Lock()
	msg := make(map[string]interface{})
	msg["type"] = "push"
	msg["address"] = pullerIpAddress
	pusherForwardTimes := 1
	if c, ok := Clients[pusherIpAddress]; ok {
		c.ConnMutex.Lock()
		c.Conn.WriteJSON(msg)
		pusherForwardTimes = c.ForwardTimes
		c.ConnMutex.Unlock()
	}
	msg["type"] = "pull"
	msg["address"] = pusherIpAddress
	if c, ok := Clients[pullerIpAddress]; ok {
		c.ConnMutex.Lock()
		c.Conn.WriteJSON(msg)
		c.ForwardTimes = pusherForwardTimes + 1
		c.ConnMutex.Unlock()
	}
	ClientsMutex.Unlock()
}

func ResourceHandler(w http.ResponseWriter, r *http.Request) {
	if ForwardsAvaliable == 0 {
		w.WriteHeader(503)
		return
	}

	if Load >= MAX_LOAD {
		w.WriteHeader(503)
		return
	}

	atomic.AddInt64(&Load, 1)
	defer func() {
		atomic.AddInt64(&Load, -1)
	}()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		Log("error", "resource", err)
		return
	}
	//ipaddress := strings.Split(r.RemoteAddr, ":")[0]
	ipaddress := r.RemoteAddr
	NewClient(ipaddress, conn)
	defer func() {
		conn.Close()
		RemoveClient(ipaddress)
	}()

	for {
		msg := make(map[string]interface{})
		err := conn.ReadJSON(&msg)
		if err != nil {
			return
		}
		//		if websocket.IsCloseError(err, WS_CLOSE_ERROR_CODES...) { //IsCloseError function has some problems to check
		//			return
		//		}

		method, ok := msg["method"].(string)
		if !ok {
			Log("error", "resource", "method missing")
			continue
		}

		switch method {
		case "getSource":
			if Clients[ipaddress].PullNum < 2 && Clients[ipaddress].ForwardTimes != 1 {
				pusherIpAddress := PeekSourceClient()
				if pusherIpAddress == "" {
					//forwardAddress := PeekSourceServer()
					c := Clients[ipaddress]
					c.ForwardTimes = 1
					c.PullNum = 1
					Clients[ipaddress] = c
					// Make forward connecttion
					resp := make(map[string]interface{})
					resp["type"] = "directPull"
					resp["address"] = PeekSourceServer()
					atomic.AddInt32(&ClientsNumDirectPull, 1)
					c.ConnMutex.Lock()
					c.Conn.WriteJSON(resp)
					c.ConnMutex.Unlock()
				} else {
					MakePeerConnection(ipaddress, pusherIpAddress)
				}
			}
		case "update":
			if pullNum, ok := msg["pullNum"].(float64); ok {
				c := Clients[ipaddress]
				c.PullNum = int(pullNum)
				Clients[ipaddress] = c
			}
			if pushNum, ok := msg["pushNum"].(float64); ok {
				c := Clients[ipaddress]
				c.PushNum = int(pushNum)
				Clients[ipaddress] = c
			}
		case "candidate": //WebRTC signal forward directly
			fallthrough
		case "offer":
			fallthrough
		case "answer":
			if address, ok := msg["address"].(string); ok {
				if c, ok := Clients[address]; ok {
					c.ConnMutex.Lock()
					msg["type"] = msg["method"]
					c.Conn.WriteJSON(msg)
					c.ConnMutex.Unlock()
				}
			}
		}
	}
}

/*
	有几点需要注意一下
	不能出现A拉B B同时拉A
	避免A拉到的两个源之间不同步过大
*/
