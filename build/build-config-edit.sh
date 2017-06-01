#!/bin/bash
bindir=$1
unamestr=`uname`

sourcedir="./src/config-edit"
if [[ ! -d "$sourcedir" ]]; then
    echo "Please run the script from project root..."
	exit -1
fi

if [[ "$unamestr" == "Linux" ]]; then
    go_os="linux"
elif [[ "$unamestr" == "Darwin" ]]; then
    go_os="darwin"
elif [[ "$unamestr" == "MSYS_NT-6.3" ]]; then
    go_os="windows"
    exe_extension=".exe"
fi

outputFile="${bindir}/config-edit${exe_extension}"
echo "Building ${outputFile}..."
CGO_ENABLED=0 GOOS=$go_os go build -o $outputFile $sourcedir
buildExitCode=$?

if [[ $buildExitCode == 0 ]]; then
    echo "Built ${outputFile} successfully"
else
    exit $buildExitCode
fi
