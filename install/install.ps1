param (
    [string] $baseurl = "https://github.com/Azure/acr-docker-credential-helper/releases/download/v0.1.0-alpha",
    [switch] $skipCleanup
)

$systemStr = (Get-WmiObject -Class Win32_ComputerSystem).SystemType
if ($systemStr -match '(x64)') {
    $arch = "amd64"
} elseif  ($systemStr -match '(x86)') {
    $arch = "x86"
} else {
    Write-Error "Unknown arch $systemStr"
    exit -1
}

if ($arch -ne "amd64") {
    Write-Error "Arch $arch is currently not supported."
    exit -1
}

$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")

if (!$isAdmin) {
    Write-Error "Please run this script as administrator"
    exit -1
}

$dockerLocation = $(where.exe docker)
if (!$dockerLocation) {
    Write-Error "Please install docker first"
    exit -1
}
$installLocation = Split-Path $dockerLocation

$tempdir = "deleteme"
$archiveFile = "docker-credential-acr-windows-${arch}.zip"

if (Test-Path $tempdir) {
    Remove-Item -Force -Recurse $tempdir
}

if (Test-Path $archiveFile) {
    Remove-Item -Force -Recurse $archiveFile    # just in case there's a directory by that name
}

Invoke-WebRequest $baseurl/$archiveFile -OutFile $archiveFile -PassThru
mkdir $tempdir
Expand-Archive -path $archiveFile -DestinationPath $tempdir

Move-Item -Force (Join-Path $tempdir "docker-credential-acr-windows*.exe") $installLocation

$configDir = Join-Path $env:UserProfile ".docker"

if (!(Test-Path $configDir)) {
    mkdir $configDir
}

$configFile = Join-Path $configDir "config.json"

if (!(Test-Path $configFile)) {
    Write-Output '{"auths":{}}' | Out-File $configFile -Encoding ASCII
}

$configEditPath = [System.IO.Path]::Combine(".", $tempdir, "config-edit.exe")
&$configEditPath "--helper" "acr-windows" "--config-file" "${configFile}"

if (!$skipCleanup) {
    Remove-Item -Force -Recurse $tempdir
    Remove-Item -Force -Recurse $archiveFile
}
