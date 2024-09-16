<img alt="abctl logo" src="https://avatars.githubusercontent.com/u/59758427?size=64" height="100%" align="left" />
<h1 align="left">abctl</h1>
Airbyte's command line tool for local Airbyte deployments.
<br clear="left"/>

---

- [Quickstart](#quickstart)
- [Commands](#commands)
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
   # install Airbyte 
   abctl local install
   # fetch the login credentials
   abctl local credentials
   ```
> [!NOTE]
> Depending on internet speed, `abctl local install` could take in excess of 15 minutes.
> 
> By default `abctl local install` configures Airbyte to accessible by all inbound requests on port `8000`.
> This typically includes access via the host's ip-address and `localhost`.
>
> If port `8000` is not available. or another port is preferred, `--port [PORT]` can be specified.

4. Login to `Airbyte`

   If `abctl local install` completed successfully, it should open a browser to http://localhost:8000
   (or to the `--host` and `--port` overrides if specified).  If this is the first time Airbyte has been
   installed you will be asked to provide an email and organization name.  To retrieve your password
   to login, run `abctl local credentials`.


# Commands

All commands and sub-commands support the following optional global flags:

| Short | Long      | Description                                                                     |
|-------|-----------|---------------------------------------------------------------------------------|
| -h    | --help    | Displays the help information, description the available options.               |
| -v    | --verbose | Enables verbose (debug) output.<br />Useful when debugging unexpected behavior. |

All commands support the following environment variables:

| Name         | Description                                     |
|--------------|-------------------------------------------------|
| DO_NOT_TRACK | Set to any value to disable telemetry tracking. |

The following commands are supported:
- [local](#local)
- [version](#version)

## local

```abctl local --help```

The local sub-commands are focused on managing the local Airbyte installation.
The following sub-commands are supports:
- [credentials](#credentials)
- [deployments](#deployments)
- [install](#install)
- [status](#status)
- [uninstall](#uninstall)
   
### credentials

```abctl local credentials```

Displays the credentials required to login to the local Airbyte installation.

> [!NOTE]
> When `abctl local install` is first executed, random `password`, `client-id`, and `client-secret`
> are generated.

Returns ths `email`, `password`, `client-id`, and `client-secret` credentials.
The `email` and  `password` are required to login to Airbyte.
The `client-id` and `client-secret` are necessary to create an [`Access Token` for interacting with the Airbyte API](https://reference.airbyte.com/reference/createaccesstoken).

For example:
```
$ abctl local credentials
{
  "password": "[RANDOM PASSWORD]",
  "client-id": "[RANDOM CLIENT-ID]",
  "client-secret": "[RANDOM CLIENT-SECRET]"
}
```

`credentials` supports the following optional flags

| Name       | Default | Description                               |
|------------|---------|-------------------------------------------|
| --email    | ""      | Changes the authentication email address. |
| --password | ""      | Changes the authentication password.      |

### deployments

```abctl local deployments```

Display kubernetes deployment information and allows for restarting a kubernetes deployment.

`deployments` supports the following optional flags

| Name      | Default | Description                       |
|-----------|---------|-----------------------------------|
| --restart | ""      | Restarts the provided deployment. | 

### install

```abctl local install```

Installs a local Airbyte instance or updates an existing installation which was initially installed by `abctl`.

> [!NOTE]
> Depending on your internet speed, `abctl local install` may take in excess of 20 minutes.

`install` supports the following optional flags:

> [!NOTE]
> An `-` in the default column indicates no value can be provided.  
> 
> These flags behave as a switch, enabled if provided, disabled if not.

| Name                | Default | Description                                                                                                                                                                                                                                            |
|---------------------|---------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| --chart-version     | latest  | Which Airbyte helm-chart version to install.                                                                                                                                                                                                           | 
| --docker-email      | ""      | Docker email address to authenticate against `--docker-server`.<br />Can also be specified by the environment-variable `ABCTL_LOCAL_INSTALL_DOCKER_EMAIL`.                                                                                             |
| --docker-password   | ""      | Docker password to authenticate against `--docker-server`.<br />Can also be specified by the environment-variable `ABCTL_LOCAL_INSTALL_DOCKER_PASSWORD`.                                                                                               |
| --docker-server     | ""      | Docker server to authenticate against.<br />Can also be specified by the environment-variable `ABCTL_LOCAL_INSTALL_DOCKER_SERVER`.                                                                                                                     |
| --docker-username   | ""      | Docker username to authenticate against `--docker-server`.<br />Can also be specified by the environment-variable `ABCTL_LOCAL_INSTALL_DOCKER_USERNAME`.                                                                                               |
| --insecure-cookies  | -       | Disables secure cookie requirements.<br />Only set if using `--host` with an insecure (non `https`) connection.                                                                                                                                        |
| --low-resource-mode | false   | Run Airbyte in low resource mode.                                                                                                                                                                                                                      |
| --host              | ""      | FQDN where the Airbyte installation will be accessed. Default is to allow for all incoming traffic on port `--port`.<br />Set this if the Airbyte installation needs a more restricted host configuration.                                             |
| --migrate           | -       | Enables data-migration from an existing docker-compose backed Airbyte installation.<br />Copies, leaving the original data unmodified, the data from a docker-compose<br />backed Airbyte installation into this `abctl` managed Airbyte installation. |
| --no-browser        | -       | Disables launching the browser when installation completes.<br />Useful to set in situations where no browser is available.                                                                                                                            |
| --port              | 8000    | Port where the Airbyte installation will be accessed.<br />Set this if port 8000 is already in use or if a different port is preferred.                                                                                                                |
| --secret            | ""      | **Can be set multiple times**.<br />Creates a kubernetes secret based on the contents of the file provided.<br />Useful when used in conjunction with `--values` for customizing installation.                                                         |
| --values            | ""      | Helm values file to further customize the Airbyte installation.                                                                                                                                                                                        |
| --volume            | ""      | **Can be set multiple times**.<br />Mounts additional volumes in the kubernetes cluster.<br />Must be in the format of `<HOST_PATH>:<GUEST_PATH>`.                                                                                                     |

### status

```abctl local status```

If an Airbyte installation exists, returns information regarding that installation.

For example:
```
$ abctl local status
Existing cluster 'airbyte-abctl' found
Found helm chart 'airbyte-abctl'
  Status: deployed
  Chart Version: 0.422.2
  App Version: 0.63.15
Found helm chart 'ingress-nginx'
  Status: deployed
  Chart Version: 4.11.1
  App Version: 1.11.1
Airbyte should be accessible via http://localhost:8000
```

### uninstall

```abctl local uninstall```

Uninstalls a local Airbyte instance.

> [!NOTE]
> The data associated with the installed Airbyte instance will not be removed.
> 
> This is done to allow Airbyte to be reinstalled at a later date with all the data preserved. 

`uninstall` supports the following optional flags:

> [!NOTE]
> An `-` in the default column indicates no value can be provided.
>
> These flags behave as a switch, enabled if provided, disabled if not.

| Name        | Default | Description                                                                    |
|-------------|---------|--------------------------------------------------------------------------------|
| --persisted | -       | Will remove all data for the Airbyte installation.<br />This cannot be undone. |


## version

```abctl version```

Displays version information about the `abctl` tool.

For example:
```
$ abctl version
version: v0.12.0
```

# Contributing

## Report an Issue
If you have found a problem with `abctl`, please open a [Github Issue](https://github.com/airbytehq/airbyte/issues/new/choose) and use the `🐛 [abctl] Report an issue with the abctl tool` template.

## Build

Install `go`
- Via [brew](https://brew.sh/)
   ```
   brew install go
   ``` 
- Via [go.dev](https://go.dev/doc/install)

This repository utilises a [Makefile](Makefile), wrapping the traditional `go` commands
used for formatting, vetting, building, and testing `go` code.

The following `make` commands are supported:

| name         | description                                                                         |
|--------------|-------------------------------------------------------------------------------------|
| `make build` | Builds the `abctl` binary, placing it in the `build` directory.                     |
| `make clean` | Removes the `build` directory.                                                      |
| `make fmt`   | [Formats the code](https://pkg.go.dev/cmd/go#hdr-Gofmt__reformat__package_sources). |
| `make test`  | Runs all the tests.                                                                 |
| `make vet`   | Runs the [vet](https://pkg.go.dev/cmd/vet) command.                                 |

