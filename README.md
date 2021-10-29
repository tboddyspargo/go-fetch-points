# Overview

`fetch-points` is a Go executable package which solves a code challenge assessment for Fetch Rewards. It provides an API as a web service to facilitate user interactions with reward points.

# Considered, But Not Implemented

It wasn't until late in my implementation that I realized that spending could be considered just another kind of transaction creation. Rather than rewind my progress to implement it with that core assumption, I decided to keep my existing implementation, which treats transactions as mutable, expendable data. My approach may not be appropriate in the real world, but I find that it addresses the core requirements of the challenge and I think it may have some performance advantages, as well.

# Usage

## Running already compiled executable

For convenience, I have uploaded a compiled binary executable to this repository. This should minimize the setup required for running the application.

1. Download the repo.
1. Open a command line prompt (bash, zsh, PowerShell, cmd, etc.)
1. Navigate to the root of the repo
1. execute the `./fetch-points` executable.
   - This will start a webserver listing on port `8080`. Use the base URL: `http://localhost:8080`.

## Running/Compiling from Source

1. [Install Go](https://golang.org/doc/install)
1. Clone this repository, or copy its files to your local machine
1. Navigate to the root of this repository
1. run the command `go run main.go` or `go build; ./fetch-points`
   - This will start a webserver listing on port `8080`. Use the base URL: `http://localhost:8080`.
1. Using your browser or an API testing application (Postman, Thunder Client, etc.) use the different routes to test the application (see below for more detail on the routes).

> NOTE: You can run the provided unit tests from the command line by navigating to the root of the project and executing `go test`.

# Routes

## /transactions (POST)

This route accepts JSON data with the following attributes:

```json
{
  "payer": "DANNON",
  "points": 1000,
  "timestamp": "2020-11-02T14:00:00Z"
}
```

The application will save the transaction and return success.

Invalid JSON or other issues with the request body will return an error status code.

## /payer-points (GET)

This route will return a JSON object representing the current total points associated with each payer. It will look like this:

```json
[
  { "payer": "DANNON", "points": 1100 },
  { "payer": "UNILEVER", "points": 200 },
  { "payer": "MILLER COORS", "points": 10000 }
]
```

If there are no points for a payer, it will not be among the results.

If there are no points available, an empty array will be returned.

## /spend (POST)

This route will allow a user to spend points that they have available, preferring older points first and without letting any point balance associated with a payer to go below zero.

A valid request will contain an object with a points attribute indicating how much they would like to spend.

```json
{
  "points": 5000
}
```

If the user has sufficient points, they will be used/removed according to the preferred order logic. A JSON object will be returned providing a summary of how many points were used from each payer.

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

By default the application will log to the same folder where the application is located into a file named with the current date in the format: `fetch-points_yyyy-mm-dd.log`.

These log messages can be used for debugging purposes, but can also be aggregated/collected and reviewed for incident response and performance monitoring.

# TODO

1. Consider having `/spend` simply add new transactions. This would require creating a separate mechanism for preventing us from having to analyze the full history of transactions all the time.
1. Backfill tests to better capture behavior. Include negative test cases.
1. Reduce unnecessary data translations (`[]Transactions` -> `PayerTotal(map[string]int32)` -> `[]PayerBalance`) to improve performance.
1. Consider using a pre-existing API library for go to reduce customized solutions.
1. Make sure to only export constants, variables, and functions that we intend/need to expose.
1. Separate methods into separate packages. `fetch-points` for main route handling logic, `fetch-points/data` for data retrieval, manipulation, and types.
1. Provide command line argument options for `--logdir` and `--port`.
1. Appropriately handle missing attributes or extraneous attributes in the request body for each route.
1. Compile documentation using `godoc` to avoid repetition.
