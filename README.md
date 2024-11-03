# Stocker

Stocker leverages data from [AZ GFD's Stocking Schedule](https://www.azgfd.com/fishing-2/where-to-fish/fish-stocking-schedule/) to provide on-demand information about fish stocking in Arizona.

## How To

Install using `go install` or cloning this repository:

```shell
go install github.com/calvinmclean/stocker/cmd/stocker@latest
```

### Get API Key

First, a Google API key is necessary to access the stocking schedule Google Sheet.

Follow [these instructions](https://developers.google.com/sheets/api/quickstart/go) from Google to setup a developer account, then get an API key instead of the Oauth credentials file.

Set the API key as the `API_KEY` environment variable or supply as a CLI flag with `--api-key`

### Run CLI

```shell
# get the last and next Winter stocking dates for the Lower Salt River and Rose Canyon Lake
stocker get -p winter -w "lower salt river" -w "rose canyon lake" --next --last
```

### Run Server

```shell
# run the server
stocker server

# use curl to get the last and next stocking dates for all CFP waters
curl 'localhost:8080/cfp?next=true&last=true&showAll=true'
```
