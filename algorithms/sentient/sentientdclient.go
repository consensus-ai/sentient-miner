package sentient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/consensus-ai/sentient-miner/clients"
)

// NewClient creates a new SentientdClient given a '[stratum+tcp://]host:port' connectionstring
func NewClient(connectionstring, pooluser string, version string) (sc clients.Client) {
	if strings.HasPrefix(connectionstring, "stratum+tcp://") {
		sc = &StratumClient{
			connectionstring: strings.TrimPrefix(connectionstring, "stratum+tcp://"),
			User: pooluser,
			Version: version,
		}
	} else {
		s := SentientdClient{}
		s.sentientdurl = "http://" + connectionstring + "/miner/header"
		sc = &s
	}
	return
}

// SentientdClient is a simple client to a sentientd
type SentientdClient struct {
	sentientdurl string
}

func decodeMessage(resp *http.Response) (msg string, err error) {
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var data struct {
		Message string `json:"message"`
	}
	if err = json.Unmarshal(buf, &data); err == nil {
		msg = data.Message
	}
	return
}

//Start does nothing
func (sc *SentientdClient) Start() {}

//SetDeprecatedJobCall does nothing
func (sc *SentientdClient) SetDeprecatedJobCall(call clients.DeprecatedJobCall) {}

//GetHeaderForWork fetches new work from the sentient daemon
func (sc *SentientdClient) GetHeaderForWork() (target []byte, header []byte, deprecationChannel chan bool, job interface{}, err error) {
	//the deprecationChannel is not used but return a valid channel anyway
	deprecationChannel = make(chan bool)

	client := &http.Client{}

	req, err := http.NewRequest("GET", sc.sentientdurl, nil)
	if err != nil {
		return
	}

	req.Header.Add("User-Agent", "Sentient-Agent")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
	case 400:
		msg, errd := decodeMessage(resp)
		if errd != nil {
			err = fmt.Errorf("Status code %d", resp.StatusCode)
		} else {
			err = fmt.Errorf("Status code %d, message: %s", resp.StatusCode, msg)
		}
		return
	default:
		err = fmt.Errorf("Status code %d", resp.StatusCode)
		return
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if len(buf) < 112 {
		err = fmt.Errorf("Invalid response, only received %d bytes", len(buf))
		return
	}

	target = buf[:32]
	header = buf[32:112]

	return
}

//SubmitHeader reports a solved header to the sentient daemon
func (sc *SentientdClient) SubmitHeader(header []byte, job interface{}) (err error) {
	req, err := http.NewRequest("POST", sc.sentientdurl, bytes.NewReader(header))
	if err != nil {
		return
	}

	req.Header.Add("User-Agent", "Sentient-Agent")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	switch resp.StatusCode {
	case 204:
	default:
		msg, errd := decodeMessage(resp)
		if errd != nil {
			err = fmt.Errorf("Status code %d", resp.StatusCode)
		} else {
			err = fmt.Errorf("%s", msg)
		}
		return
	}
	return
}
