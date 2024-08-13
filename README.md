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

> [!IMPORTANT]
> Credentials are randomly generated as part of the installation process.
>
> After installation is complete, to find your credentials run `abctl local credentials`.

1. Install `Docker`
   - [Linux](https://docs.docker.com/desktop/install/linux-install/)
   - [Mac](https://docs.docker.com/desktop/install/mac-install/)
   - [Windows](https://docs.docker.com/desktop/install/windows-install/)
   
2. Install `abctl`
   - Via [brew](https://brew.sh/)
     ```
     brew tap airbytehq/tap
     brew install abctl
     ``` 
   - Via [go install](https://go.dev/ref/mod#go-install)
     ```
     go install github.com/airbytehq/abctl@latest
     ```
   - Via [Github ](https://github.com/airbytehq/abctl/releases/latest)
3. Install `Airbyte`
   ```
   abctl local install
   abctl local credentials
   ```
> [!NOTE]
> Depending on internet speed, `abctl local install` could take in excess of 15 minutes.
> 
> By default `abctl local install` will only allow Airbyte to accessible on the host `localhost` and port `8000`.
>
> If Airbyte will be accessed outside of `localhost`, `--host [hostname]` can be specified.<br />
> If port `8000` is not available. or another port is preferred, `--port [PORT]` can be specified.

4. Login to `Airbyte`

   If `abctl local install` completed successfully, it should open a browser to http://localhost:8000
   (or to the `--host` and `--port` overrides if specified).  If this is the first time Airbyte has been
   installed you will be asked to provide an email and organization name.  To retrieve your password
   to login, run `abctl local credentials`.


# Commands

`abctl` supports the following global flags:

| short | long      | description                                                                     |
|-------|-----------|---------------------------------------------------------------------------------|
| -h    | --help    | Displays the help information, description the available options.               |
| -v    | --verbose | Enables verbose (debug) output.<br />Useful when debugging unexpected behavior. |

This tool supports the following commands and subcommand:
- local
  - install
  - status
  - uninstall
- version

## local

The `local` command supports the following sub-commands:

### credentials

### install

### status

## uninstall

## abctl version

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
