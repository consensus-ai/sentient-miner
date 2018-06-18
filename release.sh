#!/bin/bash

# error output terminates this script
set -e

if [[ -z $1 || -z $2 ]]; then
  echo "Usage: $0 PRIVATE_KEY PUBLIC_KEY VERSION"
  exit 1
fi

privkeyFile=$1
pubkeyFile=$2
version=${3:-v0.0.1}

if [ `uname -s` = Darwin ]; then
  os="osx"
else
  os="linux"
fi

# ensure we have a clean state
make clean
rm -rf release

# build binary
make

# create release
compiledBinary="sentient-miner"
kernelFile="sentient-miner.cl"
binarySuffix="${version}-${os}-amd64"
binaryName="${compiledBinary}-${binarySuffix}"
zipFile="${binaryName}.zip"

mkdir release

(
  cd release
  cp ../$compiledBinary $binaryName
  cp ../$kernelFile .
  chmod +x $binaryName
  zip -r $zipFile $binaryName $kernelFile

  openssl dgst -sha256 -sign $privkeyFile -out $zipFile.sig $zipFile
  if [[ -n $pubkeyFile ]]; then
    openssl dgst -sha256 -verify $pubkeyFile -signature $zipFile.sig $zipFile
  fi
)

echo "Done"
