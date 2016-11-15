package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type Configure struct {
	Region int    `json:"region"`
	Name   string `json:"name"`
	// Address is address for users in nearby to connect
	// eg. the server is for LAN, this address will be
	// LAN address rather WAN address
	Address    string `json:"address"`
	Coordinate string `json:"coordinate"`
}

var Config Configure

func Log(prefix string, source string, msg interface{}) {
	source_bytes := []rune(source)
	if (len(source_bytes) > 0) && source_bytes[0] >= 'a' && source_bytes[0] <= 'z' {
		source_bytes[0] = source_bytes[0] + 'A' - 'a'
	}
	source = string(source_bytes)
	log.Println("["+strings.ToUpper(prefix)+"]", source+":", msg)
}

func ReadConfig() {
	content, err := ioutil.ReadFile("./CONFIG")
	if err != nil {
		panic(err)
	}
	config := make(map[string]interface{})
	err = json.Unmarshal(content, &config)
	if err != nil {
		panic(err)
	}
	region, ok := config["region"].(float64)
	if !ok {
		panic("Config: region error")
	}
	address, ok := config["address"].(string)
	if !ok {
		panic("Config: address error")
	}
	name, ok := config["name"].(string)
	if !ok {
		panic("Config: name error")
	}
	coordinate, ok := config["coordinate"].(string)
	if !ok {
		panic("Config: coordinate error")
	}
	Config.Address = address
	Config.Coordinate = coordinate
	Config.Name = name
	Config.Region = int(region)
}

func Register() {
	reg := make(map[string]interface{})
	reg["name"] = Config.Name
	reg["address"] = Config.Address
	reg["region"] = Config.Region
	reg["forwards"] = GetForwards()
	reg_json, err := json.Marshal(reg)
	if err != nil {
		panic(err)
	}
	reader := bytes.NewReader(reg_json)
	resp, err := http.Post("http://"+Config.Coordinate+"/tracker", "application/json", reader)
	if err != nil {
		panic(err)
	}
	Log("info", "register", resp.Status)
	if resp.StatusCode != 200 {
		panic("When register, server response:" + resp.Status)
	}
}

func Timer() {
	for {
		startTime := time.Now()
		CollectInfo()
		HeartBeat()
		pastDuration := time.Now().Sub(startTime)
		if pastDuration > 60*time.Second {
			continue
		}
		time.Sleep(60*time.Second - pastDuration)
	}

}

func Initialize() {
	ReadConfig()
	ForwardsUpdate()
	Register()
	go Timer()
}

func HeartBeat() {
	reg := make(map[string]interface{})
	reg["name"] = Config.Name
	reg["address"] = Config.Address
	reg["region"] = Config.Region
	reg["load"] = Load
	reg["ctime"] = time.Now().Unix()
	reg["forwards"] = GetForwards()
	go ForwardsUpdate()
	reg_json, err := json.Marshal(reg)
	if err != nil {
		panic(err)
	}
	reader := bytes.NewReader(reg_json)
	r, err := http.NewRequest("PUT", "http://"+Config.Coordinate+"/tracker", reader)
	if err != nil {
		Log("error", "heartbeat", err)
		return
	}
	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		Log("error", "heartbeat", err)
		return
	}
	if resp.StatusCode != 200 {
		Log("error", "heartbeat", "When heartbeat, server response:"+resp.Status)
		return
	}
}

func main() {
	Initialize()
	//http.HandleFunc("/forward", ForwardHandler)
	http.HandleFunc("/debug", DebugHandler)
	http.HandleFunc("/resource", ResourceHandler)
	http.ListenAndServe(":9090", nil)
}

/*
	TODO
	To server address should be removed so that tracker server can inside NAT,
	server will not send packet to tracker or use websocket
*/
