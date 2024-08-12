<img alt="abctl logo" src="https://avatars.githubusercontent.com/u/59758427?size=64" height="100%" align="left" />
<h1 align="left">abctl</h1>
Airbyte's command line tool for local Airbyte deployments.
<br clear="left"/>

---

- [Quickstart](#quickstart)
    - [Prerequisites](#prerequisites)
    - [Install abctl](#install-abctl)
    - [Launch Airbyte](#launch-airbyte)
    - [Additional Options](#additional-options)
- [Contributing](#contributing) 

# Quickstart

> [!TIP]
> Additional documentation can be found in the [Airbyte Documentation](https://docs.airbyte.com/using-airbyte/getting-started/oss-quickstart).

1. Install `Docker`
   - [Mac instructions](https://docs.docker.com/desktop/install/mac-install/)
   - [Windows instructions](https://docs.docker.com/desktop/install/windows-install/)
   - [Linux instructions](https://docs.docker.com/desktop/install/linux-install/)
2. Install `abctl`<br />
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
  - Download the latest version from the [releases page](https://github.com/airbytehq/abctl/releases)

## Launch Airbyte
> [!Note]
> By default `abctl local install` will only allow Airbyte to accessible on the host `localhost` and port `8000`.
>
> If Airbyte will be accessed outside of `localhost`, `--host [hostname]` can be specified.<br />
> If port `8000` is not available. or another port is preferred, `--port [PORT]` can be specified.

To install and launch Airbyte locally, with default settings, run
```
abctl local install 
```

> [!IMPORTANT]
> Credentials are randomly generated as part of the installation process.
> 
> To find your credentials run `abctl local credentials`.

If `abctl local install` completed successfully, it should have opened a browser to http://localhost:8000
(or to the `--host` and `--port` overrides).  If this is the first time Airbyte has been installed
you will be asked to provide your email and organization name.  To retrieve your password to login,
run `abctl local credentials`.

# Commands

This tool supports the following commands

## local

## version

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
