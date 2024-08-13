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

All commands and sub-commands support the following optional global flags:

| short | long      | description                                                                     |
|-------|-----------|---------------------------------------------------------------------------------|
| -h    | --help    | Displays the help information, description the available options.               |
| -v    | --verbose | Enables verbose (debug) output.<br />Useful when debugging unexpected behavior. |

All commands support the following environment variables:

| name         | description                                     |
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
- [install](#install)
- [status](#status)
- [uninstall](#uninstall)
   
### credentials

```abctl local credentials```

Displays the credentials required to login to the local Airbyte installation.

> [!NOTE]
> When `abctl local install` is first executed, random `password`, `client-id`, and `client-secret`
> are generated.

Returns ths `password`, `client-id`, and `client-secret` credentials.  The `password` is the password
required to login to Airbyte. The `client-id` and `client-secret` are necessary to create an 
[`Access Token` for interacting with the Airbyte API](https://reference.airbyte.com/reference/createaccesstoken).

For example:
```
$ abctl local credentials
{
  "password": "[RANDOM PASSWORD]",
  "client-id": "[RANDOM CLIENT-ID]",
  "client-secret": "[RANDOM CLIENT-SECRET]"
}
```

### install

```abctl local install```

Installs a local Airbyte instance.

> [!NOTE]
> Depending on your internet speed, the `abctl local install` step may take in excess of 20 minutes.

Installs a local Airbyte instance or updates an existing installation that was initially installed by this tool.

`install` supports the following optional flags:

| name              | description                                                                                                                                                                                                                                            |
|-------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| --chart-version   | Which Airbyte helm-chart version to install, defaults to "latest".                                                                                                                                                                                     | 
| --docker-email    | Docker email address to authenticate against `--docker-server`.<br />Can also be specified by the environment-variable `ABCTL_LOCAL_INSTALL_DOCKER_EMAIL`.                                                                                             |
| --docker-password | Docker password to authenticate against `--docker-server`.<br />Can also be specified by the environment-variables `ABCTL_LOCAL_INSTALL_DOCKER_PASSWORD`.                                                                                              |
| --docker-server   | Docker server to authenticate against.<br />Can also be specified by the environment-variables `ABCTL_LOCAL_INSTALL_DOCKER_SERVER`.                                                                                                                    |
| --docker-username | Docker username to authenticate against `--docker-server`.<br />Can also be specified by the environment-variables `ABCTL_LOCAL_INSTALL_DOCKER_USERNAME`.                                                                                              |
| --host            | FQDN where the Airbyte installation will be accessed from, default to "localhost"<br />Set this flag if the Airbyte installation will be accessed outside of localhost.                                                                                |
| --migrate         | Enables data-migration from an existing docker-compose backed Airbyte installation.<br />Copies, leaving the original data unmodified, the data from a docker-compose<br />backed Airbyte installation into this `abctl` managed Airbyte installation. |
|                   |                                                                                                                                                                                                                                                        |
|                   |                                                                                                                                                                                                                                                        |
|                   |                                                                                                                                                                                                                                                        |
|                   |                                                                                                                                                                                                                                                        |
|                   |                                                                                                                                                                                                                                                        |
|                   |                                                                                                                                                                                                                                                        |
|                   |                                                                                                                                                                                                                                                        |
|                   |                                                                                                                                                                                                                                                        |
|                   |                                                                                                                                                                                                                                                        |


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

## version

# Contributing
If you have found a problem with `abctl`, please open a [Github Issue](https://github.com/airbytehq/airbyte/issues/new/choose) and use the `🐛 [abctl] Report an issue with the abctl tool` template.
