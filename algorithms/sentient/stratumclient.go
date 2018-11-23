package sentient

import (
	"encoding/hex"
	"errors"
	"log"
	"math/big"
	"reflect"
	"sync"

	// "golang.org/x/crypto/blake2b"
	"github.com/google/go-cmp/cmp"

	"github.com/consensus-ai/sentient-miner/clients"
	"github.com/consensus-ai/sentient-miner/clients/stratum"
)

const (
	//HashSize is the length of a sentient hash
	HashSize = 32
)

//Target declares what a solution should be smaller than to be accepted
type Target [HashSize]byte

type stratumJob struct {
	JobID        string
	PrevHash     []byte
	Coinbase1    []byte
	Coinbase2    []byte
	MerkleBranch [][]byte
	Version      string
	NBits        string
	NTime        []byte
	CleanJobs    bool
	ExtraNonce2  stratum.ExtraNonce2
}

//StratumClient is a sentient client using the stratum protocol
type StratumClient struct {
	connectionstring string
	User             string

	mutex           sync.Mutex // protects following
	stratumclient   *stratum.Client
	extranonce1     []byte
	extranonce2Size uint
	target          Target
	currentJob      stratumJob
	clients.BaseClient
}

//Start connects to the stratumserver and processes the notifications
func (sc *StratumClient) Start() {
	sc.mutex.Lock()
	defer func() {
		sc.mutex.Unlock()
	}()

	sc.currentJob.JobID = ""
	sc.DeprecateOutstandingJobs()

	sc.stratumclient = &stratum.Client{}
	//In case of an error, drop the current stratumclient and restart
	sc.stratumclient.ErrorCallback = func(err error) {
		log.Println("Error in connection to stratumserver:", err)
		sc.stratumclient.Close()
		sc.Start()
	}

	sc.subscribeToStratumDifficultyChanges()
	sc.subscribeToStratumJobNotifications()

	//Connect to the stratum server
	log.Println("Connecting to", sc.connectionstring)
	sc.stratumclient.Dial(sc.connectionstring)

	//Subscribe for mining
	//Close the connection on an error will cause the client to generate an error, resulting in te errorhandler to be triggered
	result, err := sc.stratumclient.Call("mining.subscribe", []string{"sentient-miner"})
	if err != nil {
		log.Println("ERROR Error in response from stratum:", err)
		sc.stratumclient.Close()
		return
	}
	reply, ok := result.([]interface{})
	if !ok || len(reply) < 3 {
		log.Println("ERROR Invalid response from stratum:", result)
		sc.stratumclient.Close()
		return
	}

	//Keep the extranonce1 and extranonce2_size from the reply
	if sc.extranonce1, err = stratum.HexStringToBytes(reply[1]); err != nil {
		log.Println("ERROR Invalid extrannonce1 from stratum")
		sc.stratumclient.Close()
		return
	}

	extranonce2Size, ok := reply[2].(float64)
	if !ok {
		log.Println("ERROR Invalid extranonce2_size from stratum", reply[2], "type", reflect.TypeOf(reply[2]))
		sc.stratumclient.Close()
		return
	}
	sc.extranonce2Size = uint(extranonce2Size)

	//Authorize the miner
	go func() {
		result, err = sc.stratumclient.Call("mining.authorize", []string{sc.User, ""})
		if err != nil {
			log.Println("Unable to authorize:", err)
			sc.stratumclient.Close()
			return
		}
		log.Println("Authorization of", sc.User, ":", result)
	}()

}

func (sc *StratumClient) subscribeToStratumDifficultyChanges() {
	sc.stratumclient.SetNotificationHandler("mining.set_difficulty", func(params []interface{}) {
		if params == nil || len(params) < 1 {
			log.Println("ERROR No difficulty parameter supplied by stratum server")
			return
		}
		diff, ok := params[0].(float64)
		if !ok {
			log.Println("ERROR Invalid difficulty supplied by stratum server:", params[0])
			return
		}
		log.Println("Stratum server changed difficulty to", diff)
		sc.setDifficulty(diff)
	})
}

func (sc *StratumClient) subscribeToStratumJobNotifications() {
	sc.stratumclient.SetNotificationHandler("mining.notify", func(params []interface{}) {
		log.Println("New job received from stratum server")
		if params == nil || len(params) < 9 {
			log.Println("ERROR Wrong number of parameters supplied by stratum server")
			return
		}

		sj := stratumJob{}

		sj.ExtraNonce2.Size = sc.extranonce2Size

		var ok bool
		var err error
		if sj.JobID, ok = params[0].(string); !ok {
			log.Println("ERROR Wrong job_id parameter supplied by stratum server")
			return
		}
		if sj.PrevHash, err = stratum.HexStringToBytes(params[1]); err != nil {
			log.Println("ERROR Wrong prevhash parameter supplied by stratum server")
			return
		}
		if sj.Coinbase1, err = stratum.HexStringToBytes(params[2]); err != nil {
			log.Println("ERROR Wrong coinb1 parameter supplied by stratum server")
			return
		}
		if sj.Coinbase2, err = stratum.HexStringToBytes(params[3]); err != nil {
			log.Println("ERROR Wrong coinb2 parameter supplied by stratum server")
			return
		}

		//Convert the merklebranch parameter
		merklebranch, ok := params[4].([]interface{})
		if !ok {
			log.Println("ERROR Wrong merkle_branch parameter supplied by stratum server")
			return
		}
		sj.MerkleBranch = make([][]byte, len(merklebranch), len(merklebranch))
		for i, branch := range merklebranch {
			if sj.MerkleBranch[i], err = stratum.HexStringToBytes(branch); err != nil {
				log.Println("ERROR Wrong merkle_branch parameter supplied by stratum server")
				return
			}
		}

		if sj.Version, ok = params[5].(string); !ok {
			log.Println("ERROR Wrong version parameter supplied by stratum server")
			return
		}
		if sj.NBits, ok = params[6].(string); !ok {
			log.Println("ERROR Wrong nbits parameter supplied by stratum server")
			return
		}
		if sj.NTime, err = stratum.HexStringToBytes(params[7]); err != nil {
			log.Println("ERROR Wrong ntime parameter supplied by stratum server")
			return
		}
		if sj.CleanJobs, ok = params[8].(bool); !ok {
			log.Println("ERROR Wrong clean_jobs parameter supplied by stratum server")
			return
		}
		sc.addNewStratumJob(sj)
	})
}

func (sc *StratumClient) addNewStratumJob(sj stratumJob) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	if (
		cmp.Equal(sc.currentJob.PrevHash,     sj.PrevHash)     &&
		cmp.Equal(sc.currentJob.Coinbase1,    sj.Coinbase1)    &&
		cmp.Equal(sc.currentJob.Coinbase2,    sj.Coinbase2)    &&
		cmp.Equal(sc.currentJob.MerkleBranch, sj.MerkleBranch) &&
		cmp.Equal(sc.currentJob.Version,      sj.Version)      &&
		cmp.Equal(sc.currentJob.NBits,        sj.NBits)        &&
		cmp.Equal(sc.currentJob.NTime,        sj.NTime)) {
		// If the job is actually different, only then reset the extranonce2
		// otherwise we will be submitting duplicate shares.
		sj.ExtraNonce2 = sc.currentJob.ExtraNonce2
		// log.Println("Reusing extranonce2")
	} else {
		// log.Println("Resetting extranonce2")
	}

	sc.currentJob = sj

	if sj.CleanJobs {
		sc.DeprecateOutstandingJobs()
	}
	sc.AddJobToDeprecate(sj.JobID)
}

// IntToTarget converts a big.Int to a Target.
func intToTarget(i *big.Int) (t Target, err error) {
	// Check for negatives.
	if i.Sign() < 0 {
		err = errors.New("Negative target")
		return
	}
	// In the event of overflow, return the maximum.
	if i.BitLen() > 256 {
		err = errors.New("Target is too high")
		return
	}
	b := i.Bytes()
	offset := len(t[:]) - len(b)
	copy(t[offset:], b)
	return
}

func difficultyToTarget(difficulty float64) (target Target, err error) {
	diffAsBig := big.NewFloat(difficulty)

	diffOneString := "0x00000000ffff0000000000000000000000000000000000000000000000000000"
	targetOneAsBigInt := &big.Int{}
	targetOneAsBigInt.SetString(diffOneString, 0)

	targetAsBigFloat := &big.Float{}
	targetAsBigFloat.SetInt(targetOneAsBigInt)
	targetAsBigFloat.Quo(targetAsBigFloat, diffAsBig)
	targetAsBigInt, _ := targetAsBigFloat.Int(nil)
	target, err = intToTarget(targetAsBigInt)
	return
}

func (sc *StratumClient) setDifficulty(difficulty float64) {
	target, err := difficultyToTarget(difficulty)
	if err != nil {
		log.Println("ERROR Error setting difficulty to ", difficulty)
	}
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.target = target
}

// GetHeaderForWork fetches new work over the stratum protocol
func (sc *StratumClient) GetHeaderForWork() (target []byte, header []byte, deprecationChannel chan bool, job interface{}, err error) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	job = sc.currentJob
	if sc.currentJob.JobID == "" {
		err = errors.New("No job received from stratum server yet")
		return
	}

	deprecationChannel = sc.GetDeprecationChannel(sc.currentJob.JobID)

	target = sc.target[:]

	//Create the arbitrary transaction
	en2 := sc.currentJob.ExtraNonce2.Bytes()
	err = sc.currentJob.ExtraNonce2.Increment()

	arbtx := []byte{0}
	arbtx = append(arbtx, sc.currentJob.Coinbase1...)
	arbtx = append(arbtx, sc.extranonce1...)
	arbtx = append(arbtx, en2...)
	arbtx = append(arbtx, sc.currentJob.Coinbase2...)
	// arbtxHash := blake2b.Sum256(arbtx)

	senHash := NewSenHash()
	senHash.Write(arbtx)
	arbtxHash := senHash.Sum([]byte{})

	//Construct the merkleroot from the arbitrary transaction and the merklebranches
	merkleRoot := arbtxHash
	for _, h := range sc.currentJob.MerkleBranch {
		m := append([]byte{1}[:], h...)
		m = append(m, merkleRoot[:]...)
		// merkleRoot = blake2b.Sum256(m)

		senHash := NewSenHash()
		senHash.Write(m)
		merkleRoot = senHash.Sum([]byte{})
	}

	//Construct the header
	header = make([]byte, 0, 80)
	header = append(header, sc.currentJob.PrevHash...)
	header = append(header, []byte{0, 0, 0, 0, 0, 0, 0, 0}[:]...) //empty nonce
	header = append(header, sc.currentJob.NTime...)
	header = append(header, merkleRoot[:]...)

	return
}

//SubmitHeader reports a solution to the stratum server
func (sc *StratumClient) SubmitHeader(header []byte, job interface{}) (err error) {
	sj, _ := job.(stratumJob)

	encodedExtraNonce2 := hex.EncodeToString(sj.ExtraNonce2.Bytes())
	nTime := hex.EncodeToString(sj.NTime)
	nonce := hex.EncodeToString(header[32:40])

	// sc.logSubmission(header, sj)

	sc.mutex.Lock()
	c := sc.stratumclient
	sc.mutex.Unlock()

	_, err = c.Call("mining.submit", []string{sc.User, sj.JobID, encodedExtraNonce2, nTime, nonce})
	if err != nil {
		return
	}

	return
}

func (sc *StratumClient) logSubmission(header []byte, sj stratumJob) {
	encodedExtraNonce2 := hex.EncodeToString(sj.ExtraNonce2.Bytes())
	nTime := hex.EncodeToString(sj.NTime)
	nonce := hex.EncodeToString(header[32:40])

	prevHash := hex.EncodeToString(header[:32])
	encodedExtraNonce1 := hex.EncodeToString(sc.extranonce1)

	merkleRoot := hex.EncodeToString(header[48:])
	var branches []string
	for _, h := range sc.currentJob.MerkleBranch {
		branches = append(branches, hex.EncodeToString(h))
	}
	coinbase1 := hex.EncodeToString(sc.currentJob.Coinbase1)
	coinbase2 := hex.EncodeToString(sc.currentJob.Coinbase2)
	target := hex.EncodeToString(sc.target[:])

	log.Println(
		"Submitting header.",
		"\n\tprevHash:", prevHash,
		"\n\tnonce:", nonce,
		"\n\textranonce1:", encodedExtraNonce1,
		"\n\textranonce2:", encodedExtraNonce2,
		"\n\tnTime:", nTime,
		"\n\tmerkleRoot:", merkleRoot,
		"\n\tmerkleBranches:", branches,
		"\n\tcoinbase1:", coinbase1,
		"\n\tcoinbase2:", coinbase2,
		"\n\ttarget:", target)
}
