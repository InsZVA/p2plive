package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	TRACKER_SERVER_RUNNING = iota
	TRACKER_SERVER_LOST
	TRACKER_SERVER_CLOSING

	TRACKER_HEARTBEAT_TIMEOUT = 180
)

type TrackerServer struct {
	Name          string
	Region        int
	Address       string
	Status        int
	LastHeartBeat time.Time
	Load          int
}

var Trackers = make(map[int][]TrackerServer)

// Region mutex to protect a region servers's read & write
var RegionMutex = make(map[int]sync.RWMutex)

func AddTracker(name string, region int, address string) {
	if trackers, ok := Trackers[region]; !ok || trackers == nil {
		RegionMutex[region] = sync.RWMutex{}
		mutex := RegionMutex[region]
		mutex.Lock()
		defer mutex.Unlock()
		Trackers[region] = []TrackerServer{
			TrackerServer{
				Name:          name,
				Region:        region,
				Address:       address,
				Status:        TRACKER_SERVER_RUNNING,
				LastHeartBeat: time.Now(),
				Load:          0,
			},
		}
	} else {
		mutex := RegionMutex[region]
		mutex.Lock()
		defer mutex.Unlock()
		Trackers[region] = append(Trackers[region],
			TrackerServer{
				Name:          name,
				Region:        region,
				Address:       address,
				Status:        TRACKER_SERVER_RUNNING,
				LastHeartBeat: time.Now(),
				Load:          0,
			},
		)
	}
	Log("info", "tracker", name+"@"+address+" Register")
}

func UpdateTracker(name string, region int, address string, load int, ctime int64) int {
	if trackers, ok := Trackers[region]; ok && trackers != nil {
		mutex := RegionMutex[region]
		mutex.RLock()
		defer mutex.RUnlock()
		for i, t := range trackers {
			if t.Name == name && t.Address == address {
				trackers[i].Load = load
				beattime := time.Unix(ctime, 0)
				if trackers[i].Status == TRACKER_SERVER_LOST &&
					trackers[i].LastHeartBeat.Add(TRACKER_HEARTBEAT_TIMEOUT*time.Second).Before(beattime) {
					Log("info", "tracker", name+"@"+address+" Reconnect")
					trackers[i].Status = TRACKER_SERVER_RUNNING
				}
				trackers[i].LastHeartBeat = beattime
				return 200
			}
		}
	}
	return 404
}

func DeleteTracker(name string, region int, address string) int {
	if trackers, ok := Trackers[region]; ok && trackers != nil {
		mutex := RegionMutex[region]
		mutex.RLock()
		for i, t := range trackers {
			if t.Name == name && t.Address == address {
				mutex.RUnlock()
				trackers[i].Status = TRACKER_SERVER_CLOSING // WLock may block long, change the status first
				mutex.Lock()
				trackers = append(trackers[0:i], trackers[i+1:]...)
				if len(trackers) == 0 {
					delete(RegionMutex, region)
				}
				mutex.Unlock()
				return 200
			}
		}
		mutex.RUnlock()
	}
	return 404
}

// address: address with port
func LookupRegion(address string) int { // Warning: IpV4 only
	segs := strings.Split(address, ":")
	segs = strings.Split(segs[0], ".")
	ip := make([]int, 4)
	for i := 0; i < 4; i++ {
		num, _ := strconv.Atoi(segs[i])
		ip[i] = num
	}

	if ip[0] == 127 {
		return -1
	}

	if ip[0] == 10 {
		return 0
	}

	return -2
}

/*
	TrackerHandler handle the http request about tracker,
	according to request method,
	POST: tracker register a new tracker server
	PUT: tracker server put a heartbeat and update its infomation
	GET: user get a tracker server nearby
	DELETE: tracker delete itself
*/
func TrackerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		defer r.Body.Close()
		// Read config
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			Log("error", "tracker", err)
			w.WriteHeader(500)
			return
		}
		config := make(map[string]interface{})
		err = json.Unmarshal(body, &config)
		if err != nil {
			Log("error", "tracker", err)
			w.WriteHeader(500)
			return
		}
		// Parse Config
		address, ok := config["address"].(string)
		if !ok {
			Log("error", "tracker", "address config missing")
			w.WriteHeader(403)
			return
		}
		region, ok := config["region"].(float64)
		if !ok {
			Log("error", "tracker", "region config missing")
			w.WriteHeader(403)
			return
		}
		name, ok := config["name"].(string)
		if !ok {
			Log("error", "tracker", "name config missing")
			w.WriteHeader(403)
			return
		}
		AddTracker(name, int(region), address)
		w.Write([]byte("ok"))
	case "GET":
		region := LookupRegion(r.RemoteAddr)
		if trackers, ok := Trackers[region]; ok && trackers != nil {
			mutex := RegionMutex[region]
			mutex.RLock()
			defer mutex.RUnlock()
			selected := -1
			now := time.Now()
			for i, t := range trackers {
				if t.Status != TRACKER_SERVER_RUNNING {
					continue
				}
				if t.LastHeartBeat.Add(TRACKER_HEARTBEAT_TIMEOUT * time.Second).Before(now) {
					Log("info", "tracker", t.Name+"@"+t.Address+" Lost")
					t.Status = TRACKER_SERVER_LOST
					continue
				}
				if selected == -1 || trackers[selected].Load >= t.Load {
					selected = i
				}
			}

			if selected != -1 {
				w.Write([]byte(trackers[selected].Address))
				return
			}
		}
		w.WriteHeader(404)
	case "PUT":
		defer r.Body.Close()
		// Read config
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			Log("error", "tracker", err)
			w.WriteHeader(500)
			return
		}
		config := make(map[string]interface{})
		err = json.Unmarshal(body, &config)
		if err != nil {
			Log("error", "tracker", err)
			w.WriteHeader(500)
			return
		}
		// Parse Config
		address, ok := config["address"].(string)
		if !ok {
			Log("error", "tracker", "address config missing")
			w.WriteHeader(403)
			return
		}
		region, ok := config["region"].(float64)
		if !ok {
			Log("error", "tracker", "region config missing")
			w.WriteHeader(403)
			return
		}
		name, ok := config["name"].(string)
		if !ok {
			Log("error", "tracker", "name config missing")
			w.WriteHeader(403)
			return
		}
		load, ok := config["load"].(float64)
		if !ok {
			Log("error", "tracker", "load config missing")
			w.WriteHeader(403)
			return
		}
		ctime, ok := config["ctime"].(float64)
		if !ok {
			Log("error", "tracker", "ctime config missing")
			w.WriteHeader(403)
			return
		}
		w.WriteHeader(UpdateTracker(name, int(region), address, int(load), int64(ctime)))
	case "DELETE":
		defer r.Body.Close()
		// Read config
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			Log("error", "tracker", err)
			w.WriteHeader(500)
			return
		}
		config := make(map[string]interface{})
		err = json.Unmarshal(body, &config)
		if err != nil {
			Log("error", "tracker", err)
			w.WriteHeader(500)
			return
		}
		// Parse Config
		address, ok := config["address"].(string)
		if !ok {
			Log("error", "tracker", "address config missing")
			w.WriteHeader(403)
			return
		}
		region, ok := config["region"].(float64)
		if !ok {
			Log("error", "tracker", "region config missing")
			w.WriteHeader(403)
			return
		}
		name, ok := config["name"].(string)
		if !ok {
			Log("error", "tracker", "name config missing")
			w.WriteHeader(403)
			return
		}
		w.WriteHeader(DeleteTracker(name, int(region), address))
	}
}
