<img alt="abctl logo" src="https://avatars.githubusercontent.com/u/59758427?size=64" height="100%" align="left" />
*abctl*<br />
Airbyte's command line tool for local Airbyte deployments.
<br clear="left"/>

---

- [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Install abctl](#install-abctl)
    - [Launch Airbyte](#launch-airbyte)
    - [Additional Options](#additional-options)
- [Contributing](#contributing) 

# Getting Started

## Prerequisites
- `Docker` installed
    - [Mac instructions](https://docs.docker.com/desktop/install/mac-install/)
    - [Windows instructions](https://docs.docker.com/desktop/install/windows-install/)
    - [Linux instructions](https://docs.docker.com/desktop/install/linux-install/)

## Install abctl
Pick from the following:
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

## Launch Airbyte
> [!Note]
> By default `abctl local install` will only allow Airbyte to accessible on the host `localhost` and port `8000`.
>
> If Airbyte will be accessed outside of `localhost`, `--host [hostname]` can be specified.
> If port `8000` is not available or another port is preferred, `--port [PORT]` can be specified.

To install and launch Airbyte locally, with default settings, run
```shell
abctl local install 
```

> [!IMPORTANT]
> Credentials are randomly generated as part of the installation process.
> To find your credentials run `abctl local credentials`.

After the `local install` command successfully completes, your browser should have launched and 
redirected you to http://localhost:8000 (what whatever `--host` and `--port` overrides were provided).  
You will need to provide credentials in order to access Airbyte locally, which can be found by running `abctl local credentials`.

These credentials can be changed either of the following 
- passing the `--username` and `--password` flags to the `local install` command
   ```shell
   abctl local install --username foo --password bar
   ```
- defining the environment variables `ABCTL_LOCAL_INSTALL_USERNAME` and `ABCTL_LOCAL_INSTALL_PASSWORD`
   ```shell
   ABCTL_LOCAL_INSTALL_USERNAME=foo
   ABCTL_LOCAL_INSTALL_PASSWORD=bar
   abctl local install
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
  version     Print version information

Flags:
  -h, --help      help for abctl
  -v, --verbose   enable verbose output
```
```
abctl local install --help

Usage:
  abctl local install [flags]

Flags:
      --chart-version string   specify the specific Airbyte helm chart version to install (default "latest")
  -h, --help                   help for install
  -p, --password string        basic auth password, can also be specified via ABCTL_LOCAL_INSTALL_PASSWORD (default "password")
      --port int               ingress http port (default 8000)
  -u, --username string        basic auth username, can also be specified via ABCTL_LOCAL_INSTALL_USERNAME (default "airbyte")

Global Flags:
  -v, --verbose   enable verbose output

```

## Contributing
If you have found a problem with `abctl`, please open a [Github Issue](https://github.com/airbytehq/airbyte/issues/new/choose) and use the `🐛 [abctl] Report an issue with the abctl tool` template.
