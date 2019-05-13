# Cloud Foundry Space Services CLI Plugin

## Description

Cloud Foundry Space Services CLI Plugin is a plugin for Cloud Foundry CLI tool that aims to speed-up
retrieval of list of service instances in the space targeted by Cloud Foundry CLI.

## Prerequisites

- [Download](https://docs.cloudfoundry.org/cf-cli/install-go-cli.html) and install Cloud Foundry CLI (≥6.36.1)
- [Download](https://golang.org/dl/) and install GO (≥1.11.4)

## Installation

### macOS

```bash
cf install-plugin https://github.com/micellius/cf-space-services/releases/download/v1.0.0/cf-space-services-darwin-amd64
```

### Linux

```bash
cf install-plugin https://github.com/micellius/cf-space-services/releases/download/v1.0.0/cf-space-services-linux-amd64
```

### Windows

```bash
cf install-plugin https://github.com/micellius/cf-space-services/releases/download/v1.0.0/cf-space-services-amd64.exe
```

## Upgrade

To upgrade version of Cloud Foundry Space Services CLI Plugin, you will need to uninstall previous version with command:

```bash
cf uninstall-plugin space-services-plugin
```

and then install new version as described in Installation section.

## Usage

The Cloud Foundry Space Services CLI Plugin supports the following commands:

#### ss

List <u>s</u>pace <u>s</u>ervices

<details><summary>History</summary>

| Version  | Changes                                     |
|----------|---------------------------------------------|
| `v1.0.0` | Added in `v1.0.0`                           |

</details>

```
NAME:
   ss - List space services

USAGE:
   cf ss
```

## Configuration

The configuration of the Cloud Foundry Space Services CLI Plugin is done via environment variables.
The following are supported:
  * `DEBUG=1` - enables trace logs with detailed information about currently running steps

## Limitations

Currently only first 99 service instances are shown.

## How to obtain support

If you need any support, have any question or have found a bug, please report it in the [GitHub bug tracking system](https://github.com/micellius/cf-space-services/issues). We shall get back to you.

## License

This project is licensed under the Apache Software License, v. 2 except as noted otherwise in the [LICENSE](https://github.com/micellius/cf-space-services/blob/master/LICENSE) file.
