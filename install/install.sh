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
    arm64) arch="arm64" ;;
    aarch64) arch="arm64" ;;
    *)
        echo "Unknown arch $archstr."
        exit -1
esac

if [[ "$arch" -ne "amd64" && "$arch" -ne "arm64" ]]; then
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

if [[ $HOME == "/root" ]]; then
   echo "WARNING: Home directory is /root and not pointing to a specific user. This script might not give you the result you want."
   echo "You should run \"sudo -i \${installscript}\" instead."
   read -p "Exit? (Y/n)" yn
   case $yn in
     [Yy]*) exit 1 ;;
     *) echo "Continuing...";;
   esac
fi

tempdir="deleteme"
archiveFile="docker-credential-acr-${os}-${arch}.tar.gz"
rm -f ${archiveFile}
rm -rf ${tempdir}

curl -o ${archiveFile} ${baseurl}/${archiveFile}
mkdir ${tempdir}
tar -xf ${archiveFile} -C ${tempdir}

defaultInstallLocation="/usr/local/bin"
## This is an attempt to check whether we are running the script interactively
## It can't detect when user pipes the script into bash, in that case default would still be used
case $- in
*i*)
    echo "Non-interactive shell detected. ACR Credentials Helper will be installed in default location: ${defaultInstallLocation}"
;;
*)
    echo "Please enter desired install location or press enter to install in ${defaultInstallLocation}. Note that you will need to add the install location to PATH."
    read installLocation
esac

## If user choose to install into default location, we will elevate the permission while copying the file
if [[ -z "${installLocation}" ]]; then
    installLocation=$defaultInstallLocation
    echo "Installing in default location ${installLocation}..."
    if [[ $EUID -ne 0 ]]; then
        sudoCmd=`which sudo`
        if [[ -z "$sudoCmd" ]]; then
            echo "Unable to elevate permissions."
            exit 1
        fi
        sudoOption="$sudoCmd "
    fi
else
    if [[ ! -d "${installLocation}" ]]; then
        mkdir -p ${installLocation}
    fi
fi

${sudoOption}cp ${tempdir}/docker-credential-acr-${os} ${installLocation}
${sudoOption}chmod +x ${installLocation}/docker-credential-acr-${os}

configdir="$HOME/.docker"
configFile="${configdir}/config.json"
scriptRunner=`ls -ld $HOME | awk '{print $3}'`

if [[ ! -d "${configdir}" ]]; then
    mkdir ${configdir}
fi

if [[ ! -f "${configFile}" ]]; then
    dummyConfigCreated="true"
    echo "{\"auths\":{}}" >> ${configFile}
fi

./${tempdir}/config-edit --helper acr-${os} --config-file ${configFile} --force

if [[ ! -z "${dummyConfigCreated}" ]]; then
    rm -f "${configFile}.bak"
fi

if [[ -z "$skipCleanup" ]]; then
    rm -f ${archiveFile}
    rm -rf ${tempdir}
fi
