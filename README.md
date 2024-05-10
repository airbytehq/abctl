<p align="center">
    <img alt="abctl logo" src="https://avatars.githubusercontent.com/u/59758427?size=200" height="140" />
    <h3 align="center">abctl</h3>
    <p align="center">Airbyte's command line tool for running Airbyte locally.</p>
</p>

---

- [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Installation](#installation)
    - [Launch](#launch)
    - [Additional Options](#additional-options)
- [Contributing](#contributing) 

## Getting Started

### Prerequisites
1. `Docker` installed
    - [Mac instructions](https://docs.docker.com/desktop/install/mac-install/)
    - [Windows instructions](https://docs.docker.com/desktop/install/windows-install/)
    - [Linux instructions](https://docs.docker.com/desktop/install/linux-install/)

### Installation
Do one of the following:
- Install using `brew`
   ```shell
   brew tap airbytehq/tap
   brew install abctl
   ```
- Install using `go install`
   ```shell
   go install github.com/airbytehq/abctl@latest
   ```
- Download the latest version of `abctl` from the [releases page](https://github.com/airbytehq/abctl/releases)

### Launch
To launch Airbyte locally with the default settings, simply run
```shell
abctl local install 
```

After the `local install` command completes successfully, your browser should have launched and 
redirected you to http://localhost.  You will need to provide credentials in order to access 
Airbyte locally, which default to the username `airbyte` and the password `password`.

These credentials can be changed either of the following 
- passing the `--username` and `--password` flags to the `local install` command
   ```shell
   abctl local install --username foo --password bar
   ```
- defining the environment variables `ABCTL_LOCAL_INSTALL_USERNAME` and `ABCTL_LOCAL_INSTALL_PASSWORD`
   ```shell
   ABCTL_LOCAL_INSTALL_USERNAME=foo
   ABCTL_LOCAL_INSTALL_PASSWORD=bar
   abc local install
   ```
  
### Additional Options
For additional options supported by `abctl`, pass the `--help` flag
```
abctl --help

Usage:
  abctl [command]

Available Commands:
  help        Help about any command
  local       Manages local Airbyte installations
  version     Print the version number

Flags:
      --dnt    opt out of telemetry data collection
  -h, --help   help for abctl
```
```
abctl local install --help

Usage:

  abctl local install [flags]

Flags:
  -h, --help              help for install
  -p, --password string   basic auth password, can also be specified via ABCTL_LOCAL_INSTALL_PASSWORD (default "password")
  -u, --username string   basic auth username, can also be specified via ABCTL_LOCAL_INSTALL_USERNAME (default "airbyte")

Global Flags:
      --dnt   opt out of telemetry data collection
```

## Contributing
If you have found a problem with `abctl`, please open a [Github Issue](https://github.com/airbytehq/airbyte/issues/new/choose) and use the `🐛 [abctl] Report an issue with the abctl tool` template.
