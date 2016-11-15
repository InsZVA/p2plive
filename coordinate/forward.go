package main

import (
	"net"
	"reflect"
	"time"
	"unsafe"
)

const (
	FORWARD_UPDATE_INTERVAL = 3600
)

const (
	FORWARD_READY = iota
	FORWARD_UPDATING
	FORWARD_UPDATED
)

var (
	Forwards              []string
	ForwardsUpdating      []string
	UpdateState           int
	ForwardLastUpdateTime time.Time
)

type Packet struct {
	MagicNumber  [2]byte
	Len          int32
	ForwardTimes byte
	CreateTime   int32
	Data         []byte
}

func InsertFoward(address string) {
	for _, s := range ForwardsUpdating {
		if s == address {
			return
		}
	}
	ForwardsUpdating = append(ForwardsUpdating, address)
}

/*
	UpdateForwards get all forward server address( pushing stream address)
	from all trackers. When it is updating, it will not modify current forward
	list, it stores in another list instead. When finished, ApplyUpdate() will
	copy to current list
*/
func UpdateForwards() {
	if UpdateState != FORWARD_READY {
		return
	}
	ForwardsUpdating = []string{}
	for r, trackers := range Trackers {
		mutex := RegionMutex[r]
		mutex.RLock()
		for _, t := range trackers {
			/*resp, err := http.Get("http://" + t.Address + "/forward")
			if err != nil {
				Log("error", "forward", err)
				continue
			}
			defer resp.Body.Close()
			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				Log("error", "forward", err)
				continue
			}
			var forwards []interface{}
			err = json.Unmarshal(data, &forwards)
			if err != nil {
				Log("error", "forward", err)
				continue
			}*/
			for _, s := range t.Forwards {
				InsertFoward(s)
			}
		}
		mutex.RUnlock()
	}
	UpdateState = FORWARD_UPDATED
}

func ApplyUpdate() {
	if UpdateState != FORWARD_UPDATED {
		return
	}
	Forwards = append([]string{}, ForwardsUpdating...)
	ForwardLastUpdateTime = time.Now()
	UpdateState = FORWARD_READY
}

func int322bytes(i int32) []byte {
	var b reflect.SliceHeader
	b.Cap = 4
	b.Len = 4
	b.Data = uintptr(unsafe.Pointer(&i))
	return *((*[]byte)(unsafe.Pointer(&b)))
}

func ForwardStream(data []byte, length int, address string) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		Log("error", "forward", err)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		Log("error", "forward", err)
	}
	packet := []byte{'Z', 'P'}
	packet = append(packet, int322bytes(int32(length+11))...)
	packet = append(packet, 0)
	packet = append(packet, int322bytes(int32(time.Now().Unix()))...)
	packet = append(packet, data[0:length]...)
	if len(packet) != length+11 {
		panic("ERROR LENGTH CHECK YOUR CODE")
	}
	conn.Write(packet)
}
