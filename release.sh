#!/bin/bash

# error output terminates this script
set -e

if [[ -z $1 || -z $2 ]]; then
  echo "Usage: $0 PRIVATE_KEY PUBLIC_KEY VERSION"
  exit 1
fi

privkeyFile=$1
pubkeyFile=$2
version=$3

if [ `uname -s` = Darwin ]; then
  os="osx"
else
  os="linux"
fi

# ensure we have a clean state
make clean
rm -rf release

# build binary
make release

# create release
compiledBinary="sentient-miner"
binarySuffix="${version}-${os}-amd64"
binaryName="${compiledBinary}-${binarySuffix}"
zipFile="${binaryName}.zip"

mkdir release

(
  cd release
  cp $GOPATH/bin/$compiledBinary $binaryName

  chmod +x $binaryName
  zip -r $zipFile $binaryName

  openssl dgst -sha256 -sign $privkeyFile -out $zipFile.sig $zipFile
  if [[ -n $pubkeyFile ]]; then
    openssl dgst -sha256 -verify $pubkeyFile -signature $zipFile.sig $zipFile
  fi
)

echo "Done"
