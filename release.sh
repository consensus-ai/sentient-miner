#!/bin/bash

# error output terminates this script
set -e

if [[ -z $1 || -z $2 ]]; then
  echo "Usage: $0 PRIVATE_KEY PUBLIC_KEY VERSION"
  exit 1
fi

privkeyFile=$1
pubkeyFile=$2
os=$3
arch=$4
version=$5

# ensure we have a clean state, then build binary
make clean
make dependencies
GOOS=$os GOARCH=$arch make release

# create release
compiledBinary="sentient-miner"
binarySuffix="${version}-${os}-${arch}"
binaryName="${compiledBinary}-${binarySuffix}"
zipFile="${binaryName}.zip"

mkdir release

(
  cd release
  cp $GOPATH/bin/${os}_${arch}/${compiledBinary}* $binaryName 2>/dev/null || :
  cp $GOPATH/bin/${compiledBinary}* $binaryName 2>/dev/null || :


  chmod +x $binaryName
  zip -r $zipFile $binaryName

  openssl dgst -sha256 -sign $privkeyFile -out $zipFile.sig $zipFile
  if [[ -n $pubkeyFile ]]; then
    openssl dgst -sha256 -verify $pubkeyFile -signature $zipFile.sig $zipFile
  fi
)

echo "Done"
