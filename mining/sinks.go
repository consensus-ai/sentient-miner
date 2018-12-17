package mining

import (
	"fmt"
	"log"
	"time"
	"net/http"
	"encoding/json"

	"github.com/gorilla/websocket"
)

type HashRateSink interface {
	SetCurrentHashRates(map[int]float64) error
}

type hashRateStdOutSink struct {}

type hashRateSocketSink struct {
	sockets           map[*websocket.Conn]bool
	upgrader          *websocket.Upgrader
	sendFrequency     int // Number of seconds between sends
	lastSendTimestamp int64
	lastTotalHashRate float64
}

func NewHashRateStdOutSink() *hashRateStdOutSink {
	return &hashRateStdOutSink{}
}

type CurrentHashRate struct {
	Timestamp int64   `json:"timestamp"`
	HashRate  float64 `json:"hashrate"`
}

func NewHashRateSocketSink(endpoint string, sendFrequency int) *hashRateSocketSink {
	sockets := make(map[*websocket.Conn]bool)
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	sink := &hashRateSocketSink{
		sockets: sockets,
		sendFrequency: sendFrequency,
		lastSendTimestamp: 0,
		lastTotalHashRate: 0,
		upgrader: upgrader,
	}

	http.HandleFunc("/hashrate/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sockets[conn] = true
	})

	http.HandleFunc("/hashrate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			fmt.Println("Invalid HTTP Method for GET /hashrate")
			return
		}

		currentHashRate := CurrentHashRate{
			Timestamp: sink.lastSendTimestamp,
			HashRate: sink.lastTotalHashRate,
		}

		js, err := json.Marshal(currentHashRate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})

	go func() {
		http.ListenAndServe(endpoint, nil)
	}()

	return sink
}

func (s *hashRateStdOutSink) SetCurrentHashRates(hashRates map[int]float64) error {
	fmt.Print("\r")
	var total float64
	for minerID, hashRate := range hashRates {
		fmt.Printf("%d-%.1f ", minerID, hashRate)
		total += hashRate
	}
	fmt.Printf("Total: %.2f MH/s", total)
	return nil
}

func (s *hashRateSocketSink) SetCurrentHashRates(hashRates map[int]float64) error {
	var total float64
	for _, hashRate := range hashRates {
		total += hashRate
	}

	timestamp := time.Now().Unix()
	if timestamp - s.lastSendTimestamp < int64(s.sendFrequency) {
		return nil
	}

	s.lastSendTimestamp = timestamp
	s.lastTotalHashRate = total

	for socket := range s.sockets {
		currentHashRate := CurrentHashRate{
			Timestamp: s.lastSendTimestamp,
			HashRate: s.lastTotalHashRate,
		}

		js, err := json.Marshal(currentHashRate)
		if err != nil {
			log.Printf("Websocket error: %s", err)
			continue
		}

		err = socket.WriteMessage(websocket.TextMessage, []byte(js))
		if err != nil {
			log.Printf("Websocket error: %s", err)
			socket.Close()
			delete(s.sockets, socket)
		}
	}

	return nil
}
