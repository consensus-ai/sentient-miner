# sentient-miner

GPU miner for sentent in go

All available opencl capable GPU's are detected and used in parallel.

## Binary releases

[Binaries for Windows and Linux are available in the corresponding releases](https://github.com/consensus-ai/sentient-miner/releases)


## Installation from source

### Prerequisites

* go version 1.4.2 or above (earlier version might work or not), check with `go version`
* opencl libraries on the library path
* gcc

```
go get github.com/consensus-ai/sentient-miner
```

## Run
```
sentient-miner
```

Usage:
```
  -url string
    	sentientd host and port (default "localhost:9980")
        for stratum servers, use `stratum+tcp://<host>:<port>`
  -user string
        username, most stratum servers take this in the form [payoutaddress].[rigname]
        This is optional, if solo mining sentient, this is not needed
  -I int
    	Intensity (default 28)
  -E string
        Exclude GPU's: comma separated list of devicenumbers
  -cpu
    	If set, also use the CPU for mining, only GPU's are used by default
  -v	Show version and exit
```

See what intensity gives you the best hashrate, increasing the intensity also increases the stale rate though.

## Examples

**poolmining:**
`sentient-miner -url stratum+tcp://sentientmining.com:3333 -I 28 -user 9afafe46fbd4d2fc3f6dd61ae36686a8ce3d9ddd84a8c8fa72dddb5fe09e6e61f2e2e60f974c.example`

**solomining:**
start sentientd with the miner module enabled and start sentient-miner:
`sentientd -M cghrtwm`
`sentient-miner`

## Stratum support

Stratum support is implemented as defined on https://sentientmining.com/stratum

## Developer fee

A developer fee of 1% is created by submitting 1% of the shares for my address if using the stratum protocol. The code is open source so you can simply remove that line if you want to. To make it easy for you, the exact line is https://github.com/consensus-ai/sentient-miner/blob/master/algorithms/sentient/sentientstratum.go#L307 if you do not want to support the sentient-miner development.
