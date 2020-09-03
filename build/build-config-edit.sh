#!/bin/bash
set -e
bindir=$1
goarc=$2

if [[ -z "${bindir}" ]]; then
    bindir="$PWD/bin"
fi

if [[ -z "${goarc}" ]]; then
    goarch="amd64"
fi

sourcedir="./src/config-edit"
if [[ ! -d "$sourcedir" ]]; then
    echo "Please run the script from project root..."
	exit -1
fi

export CGO_ENABLED=0
export GOARCH=$goarc
export GOPATH=$PWD
echo "Go path = $GOPATH"
for go_os in "linux" "windows" "darwin"
do
    if [[ "$go_os" == "windows" ]]; then
        exe_extension=".exe"
    else
        exe_extension=""
    fi
    outputFile="${bindir}/${go_os}/${GOARCH}/config-edit${exe_extension}"
    echo "Building ${outputFile}..."
    export GOOS=$go_os
    go build -o $outputFile $sourcedir
    buildExitCode=$?

    if [[ $buildExitCode == 0 ]]; then
        echo "Built ${outputFile} successfully"
    else
        exit $buildExitCode
    fi
done
