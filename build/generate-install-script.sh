#!/bin/bash
artifactsdir=$1

sourcedir="./install"
if [[ ! -d "$sourcedir" ]]; then
    echo "Please run the script from project root..."
	exit -1
fi

if [[ -z "${TRAVIS_TAG}" ]]; then
    echo "TRAVIS_TAG is missing... Release should be skipped..."
    TRAVIS_TAG="notags"
fi

unamestr=`uname`
if [[ "$unamestr" == "Linux" ]]; then
	osname="linux"
elif [[ "$unamestr" == "Darwin" ]]; then
	osname="osx"
else
    echo "OS $unamestr is not yet supported"
    exit -1
fi

sed -e "s#{TRAVIS_TAG}#${TRAVIS_TAG}#g" ${sourcedir}/install.sh.template > ${artifactsdir}/install-${osname}-amd64.sh
