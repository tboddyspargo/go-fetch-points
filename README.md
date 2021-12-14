# Overview

`github.com/tboddyspargo/fetch` is a Go executable module which is a submission for an interview code challenge for Fetch Rewards. It provides an API as a web service to facilitate user interactions with reward points.

# Usage

## Installing Binary

1. [Install Go](https://golang.org/doc/install). Use default `$GOROOT` and `$GOPATH` values.
1. Open a Command Prompt (bash, vsh, PowerShell, cmd)
1. Run the command `go install github.com/tboddyspargo/fetch@latest`
1. The binary `fetch` command should now exist in your `$GOROOT/bin` directory.
1. Run the command `fetch` (or `$GOROOT/bin/fetch` if `$GOROOT/bin` is not in your $PATH variable)

## Running/Compiling from Source

1. [Install Go](https://golang.org/doc/install). Use default `$GOROOT` and `$GOPATH` values.
1. Download/Clone this repository (unzip, if necessary) to your `$HOME` or `$USERPROFILE` directory.
1. Navigate to the root of this repository.
1. Run the command `go build` this will create a `fetch` binary in the root directory.
1. Execute the binary with the command `./fetch`
   - Please allow the executable to run and to be contacted by the network, if prompted.
   - This will start a webserver listing on port `8080`. Use the base URL: .
1. Using your browser or an API testing application (Postman, Thunder Client, etc.) and the base URL `http://localhost:8080`, test the application.
   - The easiest route to use to check if the application is running is to use your browser to navigate to `http://localhost:8080/health-check` which will return `{ "status": 0 }` if the application is running.

> NOTE: You can run the provided unit tests from the command line by navigating to the root of the project and executing `go test ./...`.

# Command-Line Arguments

| Parameter | Type   | Description                                                                           | Default Value | Example                    |
| --------- | ------ | ------------------------------------------------------------------------------------- | ------------- | -------------------------- |
| log-path  | String | The path to a desired log file or an existing directory where logs should be written. | ""            | --log-path /var/log/fetch/ |
| port      | String | The port to listen on.                                                                | "8080"        | --port 8080                |

# Routes

## /transaction (POST)

This route accepts JSON data with the following attributes:

```json
{
  "payer": "DANNON",
  "points": 1000,
  "timestamp": "2020-11-02T14:00:00Z"
}
```

The application will save the transaction and return a success status code and a JSON representation of the resulting object.

Invalid JSON or other issues with the request body will return an error status code and JSON.

## /payer-points (GET)

This route will return a JSON object representing the current total points associated with each payer. It will look like this:

```json
[
  { "payer": "DANNON", "points": 1100 },
  { "payer": "UNILEVER", "points": 200 },
  { "payer": "MILLER COORS", "points": 10000 }
]
```

If there are no points available, an empty array will be returned.

## /spend (POST)

This route will allow a user to spend points that they have available, preferring older points first and without letting any point balance associated with a payer to go below zero.

A valid request will contain an object with a points attribute indicating how much they would like to spend.

```json
{
  "points": 5000
}
```

If the user has sufficient points, they will be used/removed according to the preferred order logic. A JSON array will be returned providing a summary of how many points were used from each payer.

_Example return object_

```json
[
  { "payer": "DANNON", "points": -100 },
  { "payer": "UNILEVER", "points": -200 },
  { "payer": "MILLER COORS", "points": -4700 }
]
```

If the user has insufficient points, an error will be returned and no points will be used/removed.

## /health-check (GET)

This route provides a basic health check to facilitate application monitoring.

# Logging

By default the application will log to stdout and stderr as well as to a file in same folder where the application is located using the following file naming convention: `fetch-points_yyyy-mm-dd.log`.

These log messages can be used for debugging purposes, but can also be aggregated/collected and reviewed for incident response and performance monitoring.

# TODO

- [x] Consider having `/spend` simply add new transactions. This would require creating a separate mechanism for preventing us from having to analyze the full history of transactions all the time.
- [x] Backfill tests to better capture behavior. Include negative test cases.
- [ ] Consider using a pre-existing API library for go to reduce customized solutions.
- [ ] Make sure to only export constants, variables, and functions that we intend/need to expose.
- [x] Separate methods into separate packages. `fetch` for main server setup, `fetch/handler` for route handling logic, `fetch/points` for data retrieval, manipulation, and types.
- [x] Provide command line argument options for `--logdir` and `--port`.
- [x] Appropriately handle missing attributes or extraneous attributes in the request body for each route.
- [x] Compile documentation using `godoc` to avoid repetition.
- [ ] Migrate to goreadme to avoid duplicative documentation.
