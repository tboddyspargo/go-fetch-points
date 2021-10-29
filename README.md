# Overview

`fetch-points` is a Go executable package which solves a code challenge assessment for Fetch Rewards. It provides an API as a web service to facilitate user interactions with reward points.

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

Malformatted JSON or other issues with the request body will return an error status code.
