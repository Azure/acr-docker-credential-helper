#!/bin/bash
bindir=$1

if [[ -z "${bindir}" ]]; then
    bindir="$PWD/bin"
fi

sourcedir="./src/docker-credential-acr"
if [[ ! -d "$sourcedir" ]]; then
    echo "Please run the script from project root..."
	exit -1
fi

if [[ ! -z "${2}" ]]; then
	buildtags="--tags '${2}'"
fi

if [[ ! -z "${3}" ]]; then
    outputSuffix="-${3}"
fi

export CGO_ENABLED=0
export GOARCH=amd64
export GOPATH=$PWD
for go_os in "linux" "windows" "darwin"
do
    if [[ "$go_os" == "windows" ]]; then
        exe_extension=".exe"
    else
        exe_extension=""
    fi
    outputFile="${bindir}/${go_os}/docker-credential-acr-${go_os}${outputSuffix}${exe_extension}"
    echo "Building ${outputFile} ${buildtags}..."
    export GOOS=$go_os
    go build -o $outputFile ${buildtags} $sourcedir
    buildExitCode=$?

    if [[ $buildExitCode == 0 ]]; then
        echo "Built ${outputFile} successfully"
    else
        exit $buildExitCode
    fi
done
