#!/bin/bash
bindir=$1
tags=$2
unamestr=`uname`

sourcedir="./src/docker-credential-acr"
if [[ ! -d "$sourcedir" ]]; then
    echo "Please run the script from project root..."
	exit -1
fi

if [[ "$unamestr" == "Linux" ]]; then
    osname="linux"
    go_os="linux"
elif [[ "$unamestr" == "Darwin" ]]; then
    osname="osx"
    go_os="darwin"
elif [[ "$unamestr" == "MSYS_NT-6.3" ]]; then
    osname="windows"
    go_os="windows"
    exe_extension=".exe"
else
    echo "OS $unamestr is not yet supported"
    exit -1
fi

if [[ ! -z "${tags}" ]]; then
	buildtags="--tags '${tags}'"
fi

if [[ ! -z "$3" ]]; then
    outputSuffix="-${3}"
fi

outputFile="${bindir}/docker-credential-acr-${osname}${outputSuffix}${exe_extension}"
echo "Building ${outputFile} ${buildtags}..."
CGO_ENABLED=0 GOOS=$go_os go build -o $outputFile ${buildtags} $sourcedir
buildExitCode=$?

if [[ $buildExitCode == 0 ]]; then
    echo "Built ${outputFile} successfully"
else
    exit $buildExitCode
fi
