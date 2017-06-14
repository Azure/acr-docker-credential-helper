# ACR Docker login helper

This is a wrapper for Docker Credential helpers created by Azure Container Registry (ACR) team. This credential helper make use of Azure Active Directory (AAD) to obtain and maintain user's credentials.

## Prerequisites

- Docker installation is required, of course.
- [Azure CLI](https://github.com/Azure/azure-cli)

## Installation
For windows run the [powershell installation script](https://aka.ms/acr/installaad/win) in administrator mode:

`iex ([System.Text.Encoding]::ASCII.GetString((Invoke-WebRequest -Uri https://aka.ms/acr/installaad/win).Content))`

For linux, run the [bash installation script](https://aka.ms/acr/installaad/bash) as root:

`curl -L https://aka.ms/acr/installaad/bash | sudo /bin/bash`

## Usage
To login to an ACR service as followed:
    `az acr login -n <registry name>`

After that, you will be able to use docker normally. This credential helper would help maintaining your AAD access tokens.

## Troubleshooting
- Why am I getting 401 error (authentication required)?

    If you have not called `az acr login -n <registry>` to log in to your registry for extended period of time, please re-login with that command. If you find yourself having to re-login every hour or so, make sure your system clock time is correct.
