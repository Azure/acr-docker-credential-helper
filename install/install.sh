#!/bin/bash
set -e

baseurl="https://aadacr.blob.core.windows.net/acr-docker-credential-helper"
os=`uname | tr '[:upper:]' '[:lower:]'`

if [[ "$os" != "linux" && "$os" != "darwin" ]]; then
    echo "Operation System $os is not supported currently."
    exit -1
fi

archstr=`uname -m`
case "$archstr" in
    x86) arch="x86" ;;
    i?86) arch="x86" ;;
    amd64) arch="amd64" ;;
    x86_64) arch="amd64" ;;
    *)
        echo "Unknown arch $archstr."
        exit -1
esac

if [[ "$arch" -ne "amd64" ]]; then
    echo "Arch $arch is currently not supported."
    exit -1
fi

while getopts ":b:s:" opt; do
    case $opt in
        b) baseurl="$OPTARG"
        ;;
        s) skipCleanup="true"
        ;;
        \?) echo "Invalid option -$OPTARG" >&2
        ;;
    esac
done

if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root"
   exit 1
fi

if [[ $HOME == "/root" ]]; then
   echo "WARNING: Home directory is /root and not pointing to a specific user. This script might not give you the result you want."
   echo "You should run \"sudo -i \${installscript}\" instead."
   read -p "Exit? (Y)" yn
   case $yn in
     [Yy]*) exit 1 ;;
     *) echo "Continuing...";;
   esac
fi

tempdir="deleteme"
archiveFile="docker-credential-acr-${os}-${arch}.tar.gz"
rm -f ${archiveFile}
rm -rf ${tempdir}

wget ${baseurl}/${archiveFile}
mkdir ${tempdir}
tar -xf ${archiveFile} -C ${tempdir}

installLocation="/usr/local/bin"
cp ${tempdir}/docker-credential-acr-${os} $installLocation
chmod +x $installLocation/docker-credential-acr-${os}

configdir="$HOME/.docker"
configFile="${configdir}/config.json"
scriptRunner=`ls -ld $HOME | awk '{print $3}'`

if [[ ! -d "${configdir}" ]]; then
    mkdir ${configdir}
    chown ${scriptRunner} ${configdir}
fi

if [[ ! -f "${configFile}" ]]; then
    echo "{\"auths\":{}}" >> ${configFile}
    chown ${scriptRunner} ${configFile}
fi

./${tempdir}/config-edit --helper acr-${os} --config-file ${configFile}
chown ${scriptRunner} ${configFile}

if [[ -z "$skipCleanup" ]]; then
    rm -f ${archiveFile}
    rm -rf ${tempdir}
fi
