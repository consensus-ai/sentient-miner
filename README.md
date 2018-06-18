# Sentient-Miner
A GPU miner designed for mining SEN. This miner runs in a command prompt
and prints your hashrate along side the number of blocks you've mined. Most
cards will see greatly increased hashrates by increasing the value of 'I'
(default is 16, optimal is typically 20-25). Be careful with adjusting this parameter as it may crash the miner, or freeze the output.

## Install dependencies (ubuntu 16.04)

#### On ubuntu 16.04
```bash
sudo apt-get install -y ocl-icd-libopencl1 opencl-headers clinfo libcurl4-gnutls-dev
sudo ln -s /usr/lib/x86_64-linux-gnu/libOpenCL.so.1 /usr/lib/libOpenCL.so

# check OpenCL platforms
clinfo
```

#### On AWS, p2 instance
Install NVIDIA drivers first, using the guide here: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/install-nvidia-driver.html

E.g. at the time of writing, these are the steps that worked (double check the version strings when you run this, in case there's a newer version available):

```bash
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

Optionally, follow the optimization steps here:
https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/optimize_gpu.html

Here's what worked at the time of writing this:

```bash
nvidia-persistenced
nvidia-smi --auto-boost-default=0
nvidia-smi -ac 2505,875
```

#### On OSX High Sierra 10.13.5

OpenCL should already be installed. Nothing to do.

## How to Use
1) Build the miner by running `make`.

2) Make sure you have a recent version of `sentientd` installed and running.

3) Run the miner by running `./sentient-miner`. It will mine blocks until killed with Ctrl-C.

## Configuration
You can tweak the miner settings with five command-line arguments: `-I`, `-C`, `-p`, `-d`, and `-P`.
* -I controls the intensity. On each GPU call, the GPU will do 2^I hashes. The
  default value is low to prevent certain GPUs from crashing immediately at
  startup. Most cards will benefit substantially from increasing the value. The
  default is 16, but recommended (for most cards) is 20-25.
* -C controls how frequently calls to sentientd are made. Reducing it substantially
  could cause instability to the miner. Increasing it will reduce the frequency
  at which the hashrate is updated. If a low 'I' is being used, a high 'C'
  should be used. As a rule of thumb, the hashrate should only be updating a
  few times per second. The default is 30.
* -p allows you to pick a platform. Default is the first platform (indexing
  from 0).
* -d allows you to pick which device to copmute on. Default is the first device
  (indexing from 0).
* -P changes the port that the miner makes API calls to. Use this if you
  configured Sentient to be on a port other than the default. Default is 9910.

If you wanted to run the program on platform 0, device 1, with an intensity of
24, you would call `./sentient-miner -d 1 -I 24`

## Multiple GPUs
Each instance of the miner can only point to a single GPU. To mine on multiple
GPUs at the same time, you will need to run multiple instances of the miner and
point each at a different gpu. Only one instance of 'sentientd' needs to be running,
all of the miners can point to it.

It is highly recommended that you greatly increase the value of 'C' when using
multiple miners. As a rule of thumb, the hashrate for each miner should be
updating one time per [numGPUs] seconds. You should not mine with more than 6
cards at a time (per instance of sentientd).

## Notes
*    Each Sen block takes about 10 minutes to mine.
*    Once a block is mined, Sentient waits for 144 confirmation blocks before the
	 reward is added to your wallet, which takes about 24 hours.
*    Sentient currently doesn't have any mining pools.
*    Windows is poorly supported. For best results, use Linux when mining.
