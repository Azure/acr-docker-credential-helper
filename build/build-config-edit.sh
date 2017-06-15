#!/bin/bash
bindir=$1

if [[ -z "${bindir}" ]]; then
    bindir="$PWD/bin"
fi

sourcedir="./src/config-edit"
if [[ ! -d "$sourcedir" ]]; then
    echo "Please run the script from project root..."
	exit -1
fi

export CGO_ENABLED=0
export GOARCH=amd64
export GOPATH=$PWD
echo "Go path = $GOPATH"
for go_os in "linux" "windows" "darwin"
do
    if [[ "$go_os" == "windows" ]]; then
        exe_extension=".exe"
    else
        exe_extension=""
    fi
    outputFile="${bindir}/config-edit-${go_os}${exe_extension}"
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
