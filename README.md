# reg-payment-service

<img src="https://github.com/eurofurence/reg-payment-service/actions/workflows/go.yml/badge.svg" alt="test status"/>

## Overview

A backend service
Implemented in go.

Command line arguments
```-config <path-to-config-file> [-migrate-database]```

## Installation

This service uses go modules to provide dependency management, see `go.mod`.

To install required dependencies run `go mod download`

If you place this repository OUTSIDE of your gopath, `go build cmd/main.go` and `go test ./...` will download all
required dependencies by default.

In order to generate mocks, the service is using https://github.com/matryer/moq. Install the binary via `go install github.com/matryer/moq@latest`

## Open Issues and Ideas

We track open issues as GitHub issues on this repository once it becomes clear what exactly needs to be done.
