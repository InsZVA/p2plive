package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type TrackerServer struct {
	Name    string
	Region  int
	Address string
	Status  int
}

var Trackers = make(map[int][]TrackerServer)

func Log(prefix string, source string, msg interface{}) {
	log.Println("["+strings.ToUpper(prefix)+"]", source+":", msg)
}

func AddTracker(name string, region int, address string) {
	if trackers, ok := Trackers[region]; !ok || trackers == nil {
		Trackers[region] = []TrackerServer{
			TrackerServer{
				Name:    name,
				Region:  region,
				Address: address,
			},
		}
	} else {
		Trackers[region] = append(Trackers[region],
			TrackerServer{
				Name:    name,
				Region:  region,
				Address: address,
			},
		)
	}
}

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

	}
}

func main() {
	http.HandleFunc("/tracker", TrackerHandler)
}
