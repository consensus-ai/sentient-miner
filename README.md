# Sentient-Miner

GPU and CPU miner for mining SEN. This miner runs in a command prompt and prints your hashrate along side the number of blocks you've mined. Most cards will see greatly increased hashrates by increasing the value of 'I' (default is 16, optimal is typically 20-25). Be careful with adjusting this parameter as it may crash the miner, or freeze the output. All available OpenCL-capable devices are detected and used in parallel.

## Install common dependencies (Ubuntu 16.04)

#### On Ubuntu 16.04

```shell
sudo apt-get install -y ocl-icd-libopencl1 opencl-headers clinfo libcurl4-gnutls-dev
sudo ln -s /usr/lib/x86_64-linux-gnu/libOpenCL.so.1 /usr/lib/libOpenCL.so

# Check OpenCL platforms
clinfo
```

#### On AWS, p2 instance

Install NVIDIA drivers first, using the guide here: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/install-nvidia-driver.html

E.g. at the time of writing, these are the steps that worked (double check the version strings when you run this, in case there's a newer version available),

```shell
sudo apt-get update -y
sudo apt-get upgrade -y linux-aws
sudo reboot

sudo apt-get install -y gcc make linux-headers-$(uname -r)

wget http://us.download.nvidia.com/tesla/396.26/NVIDIA-Linux-x86_64-396.26.run
# select all default options
sudo /bin/sh ./NVIDIA-Linux-x86_64-396.26.run
sudo reboot

# check driver config
nvidia-smi -q | head
```

Optionally, follow these [optimization steps](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/optimize_gpu.html). Here's what worked at the time of writing,

```shell
nvidia-persistenced
nvidia-smi --auto-boost-default=0
nvidia-smi -ac 2505,875

# You might also be interested in seeing what the current clock speeds are set to:
nvidia-smi  -q -i 0 -d CLOCK
```

#### On OSX High Sierra 10.13.5

OpenCL should already be installed. Nothing to do.

## Building project

##### Prerequisites

* Drivers for your GPU

### Binary releases

##### Additional Prerequisites

Binaries for MacOS and Linux are available in the [corresponding releases](https://github.com/consensus-ai/sentient-miner/releases).

### Build from source (with Docker)

This build procedure expects the host to be using NVIDIA GPUs to run w/ GPU support (via the [NVIDIA Container Runtime for Docker](https://github.com/NVIDIA/nvidia-docker)). _If this is doesn't meet the constraints for your system (e.x. you're running an AMD GPU) you don't have to use docker to build source._

##### Additional Prerequisites

* Docker ([install instructions](https://docs.docker.com/install/))
* NVIDIA GPU drivers on host machine (e.x. [how to install on Amazon EC2 instances](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/install-nvidia-driver.html))

##### Build

```shell
git clone git@github.com:consensus-ai/sentient-miner.git
cd sentient-miner
docker build . -t sentient-miner

# To run built binary
docker run -it --rm --runtime=nvidia sentient-miner \
  /home/appuser/go/bin/sentient-miner \
  -url stratum+tcp://pool.sentient.org:3333 \
  -user 269409be5afc296549bbf5f0831e31d50ef3510b82cde37194af5867fc8f084292576e8dad85.julian
```

For development I like to run,

```shell
docker run -it --rm --runtime=nvidia sentient-miner \
  -v ../sentient-miner:/home/appuser/go/src/github.com/consensus-ai/sentient-miner:rw,z \
  bash
```

Then, once in the container, use `make`,

```shell
# To compile dependencies (only need to do if dependencies change)
make dependencies

# To compile project source
make dev
# make release

# To run built binary
$GOPATH/bin/sentient-miner \
  -url stratum+tcp://pool.sentient.org:3333 \
  -user 269409be5afc296549bbf5f0831e31d50ef3510b82cde37194af5867fc8f084292576e8dad85.julian
```

### Build from source (without Docker)

##### Optional Prerequisites

- go version manager ([gvm project page](https://github.com/moovweb/gvm))

##### Additional Required Prerequisites

* go version 1.4.2 or above (I like to manage my go versions with [gvm](https://github.com/moovweb/gvm))
* glide package manager ([install instructions](https://github.com/Masterminds/glide#install))
* gcc and make (via build-essential on Ubuntu, and Xcode command line tools on Mac)
* `sudo apt-get install pkg-config` on Ubuntu

##### Build

```shell
# If using gvm
gvm install go1.11 --prefer-binary
gvm use go1.11

mkdir ~/src
git clone git@github.com:consensus-ai/sentient-miner.git ~/src/sentient-miner

cd $GOPATH
mkdir -p src/github.com/consensus-ai/

cd $GOPATH/src/github.com/consensus-ai/
ln -s ~/src/sentient-miner .
cd $GOPATH/src/github.com/consensus-ai/sentient-miner

# To compile dependencies (only need to do if dependencies change)
make dependencies

# To compile project source
# make dev
make release

# To run built binary
$GOPATH/bin/sentient-miner
```

## Running

```shell
sentient-miner --help
```

```
Usage of sentient-miner:
  -E string
    	Exclude GPU's: comma separated list of device numbers
  -I int
    	Intensity (default 16)
  -nocpu
    	If set, don't use the CPU for mining. Uses all devices by default
  -url stratum+tcp://<host>:<port>
    	daemon or server host and port, for stratum servers, use stratum+tcp://<host>:<port> (default "localhost:9910")
  -user string
    	username, most stratum servers take this in the form [payoutaddress].[rigname] (default "payoutaddress.rigname")
  -v	Show version and exit
```

See what intensity gives you the best hashrate, increasing the intensity also increases the stale rate though.

## Release

From your host machine,
```bash
gvm use go1.11 # If you're using GVM
cd $GOPATH/src/github.com/consensus-ai/sentient-miner
./release.sh $(pwd)/private.pem $(pwd)/public.pem <platform> <architecture> <version-number>
# E.x. ./release.sh $(pwd)/private.pem $(pwd)/public.pem darwin amd64 1.0
```

Or through docker (e.x. to build Linux binary or Windows binary from macOS)
```bash
docker build -t sentient-miner .
docker run -it --rm -v $(pwd)/:/home/appuser/go/src/github.com/consensus-ai/sentient-miner sentient-miner bash
./release.sh $(pwd)/private.pem $(pwd)/public.pem <platform> <architecture> <version-number>
# E.x. CC_FOR_TARGET=x86_64-w64-mingw32-gcc ./release.sh $(pwd)/private.pem $(pwd)/public.pem windows amd64 1.0

# Binaries will be on host machine under sentient-miner/release/
```

## Examples

##### Solo mining

Start sentientd with the miner module enabled and start sentient-miner,

```shell
sentientd -M cghrtwm
sentient-miner
```

##### Pool mining (via Stratum)

```shell
sentient-miner \
  -url stratum+tcp://pool.sentient.org:3333 \
  -user 269409be5afc296549bbf5f0831e31d50ef3510b82cde37194af5867fc8f084292576e8dad85.julian
```

## Stratum support

Stratum support is implemented as defined on https://pool.sentient.org/stratum.
