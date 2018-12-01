package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	zmq "github.com/pebbe/zmq4"
	"github.com/robvanmieghem/go-opencl/cl"
	"github.com/consensus-ai/sentient-miner/algorithms/sentient"
	"github.com/consensus-ai/sentient-miner/mining"
)

// Version is the released version string of sentient-miner
var Version = "v0.1.1"

var intensity = 16
var devicesTypesForMining = cl.DeviceTypeAll

func main() {
	log.SetOutput(os.Stdout)
	printVersion := flag.Bool("v", false, "Show version and exit")
	noCPU := flag.Bool("nocpu", false, "If set, don't use the CPU for mining. Uses all devices by default")
	flag.IntVar(&intensity, "I", intensity, "Intensity")
	host := flag.String("url", "localhost:9910", "daemon or server host and port, for stratum servers, use `stratum+tcp://<host>:<port>`")
	pooluser := flag.String("user", "payoutaddress.rigname", "username, most stratum servers take this in the form [payoutaddress].[rigname]")
	excludedGPUs := flag.String("E", "", "Exclude GPU's: comma separated list of device numbers")
	flag.Parse()

	if *printVersion {
		fmt.Println("sentient-miner version", Version)
		os.Exit(0)
	}

	if *noCPU {
		devicesTypesForMining = cl.DeviceTypeGPU
	}
	globalItemSize := int(math.Exp2(float64(intensity)))

	platforms, err := cl.GetPlatforms()
	if err != nil {
		log.Panic(err)
	}

	clDevices := make([]*cl.Device, 0, 4)
	for _, platform := range platforms {
		log.Println("Platform", platform.Name())
		platormDevices, err := cl.GetDevices(platform, devicesTypesForMining)
		if err != nil {
			log.Println(err)
		}
		log.Println(len(platormDevices), "device(s) found:")
		for i, device := range platormDevices {
			log.Println(i, "-", device.Type(), "-", device.Name())
			clDevices = append(clDevices, device)
		}
	}

	if len(clDevices) == 0 {
		log.Println("No suitable opencl devices found")
		os.Exit(1)
	}

	//Filter the excluded devices
	miningDevices := make(map[int]*cl.Device)
	for i, device := range clDevices {
		if deviceExcludedForMining(i, *excludedGPUs) {
			continue
		}
		miningDevices[i] = device
	}

	nrOfMiningDevices := len(miningDevices)
	var hashRateReportsChannel = make(chan *mining.HashRateReport, nrOfMiningDevices*10)

	var miner mining.Miner
	log.Println("Starting sentient mining")
	c := sentient.NewClient(*host, *pooluser, Version)

	miner = &sentient.Miner{
		ClDevices:       miningDevices,
		HashRateReports: hashRateReportsChannel,
		Intensity:       intensity,
		GlobalItemSize:  globalItemSize,
		Client:          c,
	}
	miner.Mine()

	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		log.Println("Unable to create zmq socket for publishing hashrates")
	}
	defer socket.Close()

	socketEndpoint, ok := os.LookupEnv("SENTIENT_MINER_CURRENT_HASHRATE_ENDPOINT")
	if !ok {
		socketEndpoint = "tcp://127.0.0.1:5555"
	}
	hashRatesSocketFrequency := 1 // 1 second
	if socketFrequencyStr, ok := os.LookupEnv("SENTIENT_MINER_CURRENT_HASHRATE_FREQUENCY"); ok {
		if i64, err := strconv.ParseInt(socketFrequencyStr, 10, 32); err == nil {
			hashRatesSocketFrequency = int(i64)
		}
	}
	hashRatesLogPath, ok := os.LookupEnv("SENTIENT_MINER_HASHRATES_LOG_PATH")
	if !ok {
		hashRatesLogPath = "./hashrates.log"
	}
	hashRatesLogFrequency := 10 * 60 // 10 minutes
	if logFrequencyStr, ok := os.LookupEnv("SENTIENT_MINER_HASHRATES_LOG_FREQUENCY"); ok {
		if i64, err := strconv.ParseInt(logFrequencyStr, 10, 32); err == nil {
			hashRatesLogFrequency = int(i64)
		}
	}
	maxLogLines := 10000
	if maxLogLinesStr, ok := os.LookupEnv("SENTIENT_MINER_HASHRATES_LOG_MAX_LINES"); ok {
		if i64, err := strconv.ParseInt(maxLogLinesStr, 10, 32); err == nil {
			maxLogLines = int(i64)
		}
	}

	socket.Bind(socketEndpoint)
	socket.SetLinger(0)
	stdOutSink := mining.NewHashRateStdOutSink()
	socketSink := mining.NewHashRateSocketSink(socket, hashRatesSocketFrequency)
	loggerSink := mining.NewHashRateLoggerSink(hashRatesLogPath, hashRatesLogFrequency, maxLogLines)

	//Start printing out the hashrates of the different gpu's
	hashRateReports := make(map[int]float64)
	for {
		//No need to print at every hashreport, we have time
		for i := 0; i < nrOfMiningDevices; i++ {
			report := <-hashRateReportsChannel
			hashRateReports[report.MinerID] = report.HashRate
		}

		stdOutSink.SetCurrentHashRates(hashRateReports)
		socketSink.SetCurrentHashRates(hashRateReports)
		loggerSink.SetCurrentHashRates(hashRateReports)
	}
}

//deviceExcludedForMining checks if the device is in the exclusion list
func deviceExcludedForMining(deviceID int, excludedGPUs string) bool {
	excludedGPUList := strings.Split(excludedGPUs, ",")
	for _, excludedGPU := range excludedGPUList {
		if strconv.Itoa(deviceID) == excludedGPU {
			return true
		}
	}
	return false
}
