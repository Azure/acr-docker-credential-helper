#!/bin/bash
buildImageName="acr-cred-helper-build-img"
buildContainerName="acr-cred-helper-build"
if [ "$(uname -m)" == "aarch64" ] || [ "$(uname -m)" == "arm64" ]
then
   buildGoArch="arm64"
else
   buildGoArch="amd64"
fi

if [[ "$(docker images -q ${buildImageName} 2> /dev/null)" == "" ]]; then
    docker rmi -f ${buildImageName}
fi

set -e
./build/build-clean.sh bin artifacts

docker build -t ${buildImageName} .
docker run --name ${buildContainerName} -e GOARCH=$buildGoArch ${buildImageName}
docker cp ${buildContainerName}:/build-root/artifacts artifacts

docker rm ${buildContainerName}
docker rmi ${buildImageName}
