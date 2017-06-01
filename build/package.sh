#!/bin/bash
bindir=$1
artifactsdir=$2
unamestr=`uname`

if [[ -z "${TRAVIS_TAG}" ]]; then
    echo "TRAVIS_TAG is missing... Release should be skipped..."
    TRAVIS_TAG="notags"
fi

if [[ "$unamestr" == "Linux" ]]; then
	osname="linux"
elif [[ "$unamestr" == "Darwin" ]]; then
	osname="osx"
else
    echo "OS $unamestr is not yet supported"
    exit -1
fi

if [[ ! -d $artifactsdir ]]; then
    mkdir $artifactsdir
fi

pkgFile="docker-credential-acr-${osname}-amd64.tar.gz"
pushd ${bindir}
tar czf ${pkgFile} *
popd
mv ${bindir}/$pkgFile ${artifactsdir}

echo "Packaged in ${pkgFile}"
