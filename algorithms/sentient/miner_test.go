package sentient

import (
	"bytes"
	"log"
	"math"
	"testing"

	"github.com/robvanmieghem/go-opencl/cl"
	"github.com/consensus-ai/sentient-miner/mining"
)

var provenSolutions = []struct {
	height          int
	hash            string
	workHeader      []byte
	offset          int
	submittedHeader []byte
}{
	{
		// Fake solution
		height:          1,
		hash:            "00000000000006418b86014ff54b457f52665b428d5af57e80b0b7ec84c706e5",
		// Nonce has been replaced by the target [32, 40); done outside tests as well in miner.go
		// Header: <parent block ID (32)><target (8; was nonce, but updated in miner.go)><timestamp (8)><merkle root (32)>
		workHeader:      []byte{
			0, 0, 0, 0, 0, 0, 26, 158, 25, 209, 169, 53, 113, 22, 90, 11, 72, 7, 222, 103, 247, 244, 163, 156, 158, 5, 53, 126, 186, 215, 88, 48,
			// 45, 32, 0, 0, 0, 0, 0, 0,
			//>>> struct.unpack('<Q', bytearray([255, 255, 255, 255, 255, 255, 0, 0,]))
			//(281474976710655,)
			255, 255, 255, 255, 255, 255, 0, 0,
			20, 25, 103, 87, 0, 0, 0, 0, 218, 189, 84, 137, 247, 169, 197, 113, 213, 120, 125, 148, 92, 197, 47, 212, 250, 153, 114, 53, 199, 209, 183, 97, 28, 242, 206, 120, 191, 202, 34, 9},
		offset:          5 * int(math.Exp2(float64(16))),
		// Header: <parent block ID (32)><nonce (8)><timestamp (8)><merkle root (32)>
		submittedHeader: []byte{
			0, 0, 0, 0, 0, 0, 26, 158, 25, 209, 169, 53, 113, 22, 90, 11, 72, 7, 222, 103, 247, 244, 163, 156, 158, 5, 53, 126, 186, 215, 88, 48,
			// 88, 47, 107, 95, 0, 0, 0, 0,
			132, 26, 5, 0, 0, 0, 0, 0,
			20, 25, 103, 87, 0, 0, 0, 0, 218, 189, 84, 137, 247, 169, 197, 113, 213, 120, 125, 148, 92, 197, 47, 212, 250, 153, 114, 53, 199, 209, 183, 97, 28, 242, 206, 120, 191, 202, 34, 9},
	},
}

func TestMine(t *testing.T) {
	platforms, err := cl.GetPlatforms()
	if err != nil {
		log.Panic(err)
	}

	var clDevice *cl.Device
	for _, platform := range platforms {
		platormDevices, err := cl.GetDevices(platform, cl.DeviceTypeAll)
		if err != nil {
			log.Fatalln(err)
		}
		for _, device := range platormDevices {
			log.Println(device.Type(), "-", device.Name())
			clDevice = device
		}
	}

	workChannel := make(chan *miningWork, len(provenSolutions)+1)

	for _, provenSolution := range provenSolutions {
		workChannel <- &miningWork{provenSolution.workHeader, provenSolution.offset, nil}
	}
	close(workChannel)
	var hashRateReportsChannel = make(chan *mining.HashRateReport, len(provenSolutions)+1)
	validator := newSubmittedHeaderValidator(len(provenSolutions))
	miner := &singleDeviceMiner{
		ClDevice:          clDevice,
		MinerID:           0,
		HashRateReports:   hashRateReportsChannel,
		GlobalItemSize:    int(math.Exp2(float64(16))),
		miningWorkChannel: workChannel,
		Client:            validator,
	}
	miner.mine()
	validator.validate(t)
}

func newSubmittedHeaderValidator(capacity int) (v *submittedHeaderValidator) {
	v = &submittedHeaderValidator{}
	v.submittedHeaders = make(chan []byte, capacity)
	return
}

type submittedHeaderValidator struct {
	submittedHeaders chan []byte
}

//SubmitHeader stores solved so they can later be validated after the testrun
func (v *submittedHeaderValidator) SubmitHeader(header []byte, job interface{}) (err error) {
	v.submittedHeaders <- header
	return
}

func (v *submittedHeaderValidator) validate(t *testing.T) {
	if len(v.submittedHeaders) != len(provenSolutions) {
		t.Fatal("Wrong number of headers reported")
	}
	for _, provenSolution := range provenSolutions {
		submittedHeader := <-v.submittedHeaders
		if !bytes.Equal(submittedHeader, provenSolution.submittedHeader) {
			t.Error(
				"Mismatch\nExpected header:\t", provenSolution.submittedHeader,
				"\nSubmitted header:\t", submittedHeader)
		}
	}
}
