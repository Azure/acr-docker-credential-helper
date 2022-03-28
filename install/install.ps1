param (
    [string] $baseurl = "https://aadacr.blob.core.windows.net/acr-docker-credential-helper",
    [switch] $skipCleanup
)

Write-Host "ACR Credential Helper currently does not support Windows Credential Manager because Windows Credential Manager only support saving tokens with less than 2.5KB blob size."
Write-Host "1. A json file will be used to store all your credentials."
Write-Host "2. You will have to re-login to any existing Docker registry after the installation."
$acceptFileStore = Read-Host "Continue? [Y/n]"

if (!$acceptFileStore.ToLower().StartsWith("y")) {
    Write-Error "User aborted."
    break
}

$systemStr = (Get-WmiObject -Class Win32_ComputerSystem).SystemType
if ($systemStr -match '(x64)') {
    $arch = "amd64"
} elseif  ($systemStr -match '(x86)') {
    $arch = "x86"
} else {
    Write-Error "Unknown arch $systemStr"
    break
}

if ($arch -ne "amd64") {
    Write-Error "Arch $arch is currently not supported."
    break
}

$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")

if (!$isAdmin) {
    Write-Error "Please run this script as administrator"
    break
}

$dockerLocation = $(where.exe docker)
if (!$dockerLocation) {
    Write-Error "Cannot find docker in path. Ensure it's installed and that its path is accessible."
    break
}
if($dockerLocation -is [System.Array]) {
    $dockerLocation = $dockerLocation[0]
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
    $dummyConfigCreated = $true
    Write-Output '{"auths":{}}' | Out-File $configFile -Encoding ASCII
}

$configEditPath = [System.IO.Path]::Combine(".", $tempdir, "config-edit.exe")
&$configEditPath "--helper" "acr-windows" "--config-file" "${configFile}"

if ($dummyConfigCreated) {
    Remove-Item -Force "${configFile}.bak"
}

if (!$skipCleanup) {
    Remove-Item -Force -Recurse $tempdir
    Remove-Item -Force -Recurse $archiveFile
}
