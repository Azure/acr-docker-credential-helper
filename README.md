# ACR Docker login helper

The ACR Docker Credential Helper allows users to sign-in to the Azure Container Registry service using their Azure Active Directory (AAD) credentials. This credential helper is in charge of ensuring that the stored credentials are valid, and when required it also renews the credentials for a repository.

For now, this credential helper works in tandem with the Azure CLI, which is required in order to initiate the credential flow. Once you've successfully logged in to your container registry with the Azure CLI, the credential helper administers the life cycle of your locally stored credential.

## Prerequisites

- [Docker](https://www.docker.com/)
- [Azure CLI](https://github.com/Azure/azure-cli)

## Installation
For Windows, run the [powershell installation script](https://aka.ms/acr/installaad/win) in administrator mode:

`iex ([System.Text.Encoding]::UTF8.GetString((Invoke-WebRequest -Uri https://aka.ms/acr/installaad/win).Content))`

For Linux and macOS, run the [bash installation script](https://aka.ms/acr/installaad/bash) as root:

`curl -L https://aka.ms/acr/installaad/bash | sudo /bin/bash`

## Usage
After installing the ACR Docker Credential Helper, login to an Azure Container Registry using the Azure CLI:
    `az acr login -n <registry name>`

After that, you will be able to use docker normally. This credential helper will help maintaining your credentials.

## Troubleshooting
### Getting 401 (authentication required)

If you have not called `az acr login -n <registry>` to log in to your registry for an extended period of time and you get a 401 error, please log in again. If you find yourself having to log in every hour or so, make sure your computer clock is set to the correct time.
