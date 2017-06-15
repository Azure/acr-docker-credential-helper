#!/bin/bash
set -e
binroot=$1
artifactsdir=$2

if [[ -z "${TRAVIS_TAG}" ]]; then
    echo "TRAVIS_TAG is missing... Release should be skipped..."
    TRAVIS_TAG="notags"
fi

if [[ ! -d $binroot ]]; then
    echo "Please run the script from project root..."
	exit -1
fi

if [[ ! -d $artifactsdir ]]; then
    mkdir $artifactsdir
fi

for osname in $(ls $binroot) ; do
    for arch in $(ls "$binroot/$osname") ; do
        bindir="$binroot/$osname/$arch"
        if [[ -d $bindir ]]; then
            echo "Packaging for ${osname}-${arch}"
            pkgName="docker-credential-acr-${osname}-${arch}"
            pushd $bindir
            if [[ "$osname" == "windows" ]]; then
                pkgFile="${pkgName}.zip"
                zip ${pkgFile} *
            else
                pkgFile="${pkgName}.tar.gz"
                tar czf ${pkgFile} *
            fi
            popd
            mv ${bindir}/$pkgFile ${artifactsdir}
            echo "Packaged in ${pkgFile}"
        fi
    done
done
