package mining

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"bufio"
	"net/http"

	"github.com/gorilla/websocket"
  "github.com/natefinch/atomic"
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

type hashRateLoggerSink struct {
	filePath         string
	logFrequency     int // Number of seconds between logs
	lastLogTimestamp int64
	maxLogLines      int // Max number of lines the log file is allowed to grow to
}

func NewHashRateStdOutSink() *hashRateStdOutSink {
	return &hashRateStdOutSink{}
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
			fmt.Println(err)
			return
		}

		sockets[conn] = true
	})

	http.HandleFunc("/hashrate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			fmt.Println("Invalid HTTP Method for GET /hashrate")
			return
		}

		message := fmt.Sprintf("%d,%.6f", sink.lastSendTimestamp, sink.lastTotalHashRate)
		fmt.Fprintln(w, message)
	})

	go func() { http.ListenAndServe(endpoint, nil) }()

	return sink
}

func NewHashRateLoggerSink(filePath string, logFrequency int, maxLogLines int) *hashRateLoggerSink {
	return &hashRateLoggerSink{
		filePath: filePath,
		logFrequency: logFrequency,
		lastLogTimestamp: 0,
		maxLogLines: maxLogLines,
	}
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
	message := fmt.Sprintf("%.6f", total)
	// _, err := s.socket.Send(message, 0)

	for socket := range s.sockets {
		// mu.Lock()
		err := socket.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Printf("Websocket error: %s", err)
			socket.Close()
			delete(s.sockets, socket)
		}
	}

	return nil
}

func (s *hashRateLoggerSink) SetCurrentHashRates(hashRates map[int]float64) error {
	var total float64
	for _, hashRate := range hashRates {
		total += hashRate
	}

	timestamp := time.Now().Unix()
	if timestamp - s.lastLogTimestamp < int64(s.logFrequency) {
		return nil
	}
	s.lastLogTimestamp = timestamp

	lines, err := readLines(s.filePath)
	if err != nil {
		log.Println("Unable to read hashrates log")
		return err
	}

	offset := max(len(lines) - s.maxLogLines - 1, 0)
	newLogLine := fmt.Sprintf("%d,%.6f", timestamp, total)
	lines = append(lines, newLogLine)
	lines = lines[offset:]
	if err := atomicWriteLines(lines, s.filePath); err != nil {
		log.Println("Unable to write hashrates log")
		return err
	}
	return nil
}

func readLines(path string) ([]string, error) {
  file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
  if err != nil {
    return nil, err
  }
  defer file.Close()

  var lines []string
  scanner := bufio.NewScanner(file)
	// scanner.Split(bufio.ScanLines)
  for scanner.Scan() {
    lines = append(lines, strings.TrimSpace(scanner.Text()))
  }
  return lines, scanner.Err()
}

func atomicWriteLines(lines []string, path string) error {
  reader := strings.NewReader(strings.Join(lines, "\n"))
  return atomic.WriteFile(path, reader)
}

func writeLines(lines []string, path string) error {
  file, err := os.Create(path)
  if err != nil {
    return err
  }
  defer file.Close()

  w := bufio.NewWriter(file)
  for _, line := range lines {
    fmt.Fprintln(w, line)
  }
  return w.Flush()
}

func max(x, y int) int {
    if x > y {
        return x
    }
    return y
}
