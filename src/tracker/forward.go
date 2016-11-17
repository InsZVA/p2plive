package main

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	FORWARD_RUNNING = iota
	FORWARD_LOST

	FORWARD_TIMEOUT = 10
)

type ForwardServer struct {
	// Coordinate Push stream to this address
	PushStreamAddress string
	// Clients Pull stream from this address(websocket&http load request)
	PullStreamAddress string

	Load   int
	Status int
}

var (
	Forwards             = []ForwardServer{}
	ConfigFileLastModify time.Time
	ForwardsMutex        = sync.Mutex{}
	ForwardsAvaliable    = 0
)

func ForwardsUpdate() {
	file, err := os.Open("./FORWARDS")
	defer file.Close()
	if err != nil {
		Log("error", "forward", err)
		return
	}
	if fs, err := file.Stat(); err != nil && fs.ModTime() == ConfigFileLastModify {
		return
	} else {
		ConfigFileLastModify = fs.ModTime()
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		Log("error", "forward", err)
		return
	}
	forwards := make([]interface{}, 0)
	updated := []ForwardServer{}
	err = json.Unmarshal(content, &forwards)
	if err != nil {
		Log("error", "forward", err)
		return
	}
	for _, f := range forwards {
		fm, ok := f.(map[string]interface{})
		if !ok {
			Log("error", "forward", "config error")
			return
		}
		push_a, ok := fm["push"].(string)
		if !ok {
			Log("error", "forward", "config error")
			return
		}
		pull_a, ok := fm["pull"].(string)
		if !ok {
			Log("error", "forward", "config error")
			return
		}
		updated = append(updated, ForwardServer{
			PushStreamAddress: push_a,
			PullStreamAddress: pull_a,
		})
	}
	ForwardsMutex.Lock()
	Forwards = append([]ForwardServer{}, updated...)
	ForwardsMutex.Unlock()
}

/*
	ForwardHandler handle the http request about forward server
	GET: list the push address of forwards to coordinate
*/
func ForwardHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ForwardsMutex.Lock()
		defer ForwardsMutex.Unlock()
		address := []string{}
		for _, f := range Forwards {
			if f.Status != FORWARD_RUNNING {
				continue
			}
			address = append(address, f.PushStreamAddress)
		}
		jsondata, err := json.Marshal(address)
		if err != nil {
			Log("error", "forward", err)
			w.WriteHeader(500)
			return
		}
		w.Write(jsondata)
		go ForwardsUpdate()
	default:
		w.WriteHeader(403)
	}
}

func CollectInfo() {
	ForwardsMutex.Lock()
	defer ForwardsMutex.Unlock()
	forwardsAvaliable := 0
	for _, f := range Forwards {
		c := http.Client{
			Transport: &http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					deadline := time.Now().Add(FORWARD_TIMEOUT * time.Second)
					c, err := net.DialTimeout(netw, addr, time.Second*20)
					if err != nil {
						return nil, err
					}
					c.SetDeadline(deadline)
					return c, nil
				},
			},
		}
		resp, err := c.Get("http://" + f.PullStreamAddress + "/load")
		if err != nil {
			Log("error", "collect", err)
			f.Status = FORWARD_LOST
			continue
		}
		defer resp.Body.Close()
		response, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			Log("error", "collect", err)
			continue
		}
		load, err := strconv.Atoi(string(response))
		if err != nil {
			Log("error", "collect", err)
			continue
		}
		f.Load = load
		f.Status = FORWARD_RUNNING
		forwardsAvaliable++
	}
	ForwardsAvaliable = forwardsAvaliable
}
