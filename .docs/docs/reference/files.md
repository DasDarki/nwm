---
title: Files and folders
---

# Files and folders

## nwm home

Everything nwm stores lives in `~/.nwm`, or in `NWM_HOME` if set.

| Path | Content |
|---|---|
| `state.toml` | Registered servers (name, folder, branch, dates), unsafe library approvals, the global `allow_unsafe_libs` setting |
| `servers/<name>/` | Default location of servers created by nwm |
| `secrets/<name>.toml` | Secrets of a server, mode `0600` |
| `logs/<name>.log` | Output of the last background start |
| `run/<name>.json` | Process id of a running server |
| `steamcmd/` | SteamCMD downloaded by nwm |
| `cache/registry/index.json` | Cached registry index (one day) |
| `cache/store-index.json` | Cached nanos world store catalog for completion (one day) |
| `cache/recipes/` | Checkouts and downloads of remote recipes |
| `cache/modules/` | Downloaded C module files |
| `cache/nanossharp/` | NanosSharp installer |
| `cache/update-check.json` | Time and result of the last update check |

## Server folder

| Path | Owner | Content |
|---|---|---|
| `Config.toml` | nanos world, nwm | Server configuration |
| `nwm.toml` | nwm | Applied recipe, launch arguments, services |
| `.env` | nwm, you | Environment variables for the server process, mode `0600` |
| `Packages/`, `Assets/` | nanos world, nwm | Installed packages and asset packs |
| `NanosSharp.sh`, `NanosSharp/` | NanosSharp | Present when NanosSharp is installed |

### nwm.toml

```toml
args = ['--log_level', '2']

[recipe]
source = '@dasdarki/roleplay@^1'
name = '@dasdarki/roleplay'
version = 'v1.2.0'
commit = 'a3058d40d148f2ebf08c224de2bb4e4d1c90fb25'
applied_at = 2026-11-05T21:10:49+01:00

[recipe.variables]
port = '7790'

[[service]]
name = 'postgres'
image = 'postgres:17'
ports = ['5432']
volumes = ['postgres-data:/var/lib/postgresql/data']
ready = ['pg_isready', '-h', '127.0.0.1', '-U', 'nanos', '-d', 'nanos']

[service.env]
POSTGRES_PASSWORD = '${secret.postgres_password}'
```

You can edit it by hand; `nwm service add` and `nwm recipe apply` update it.

## Docker objects

| Object | Name |
|---|---|
| Container | `nwm-<server>-<service>` |
| Named volume | `nwm-<server>-<volume>` |
| Labels | `nwm.server`, `nwm.service`, `nwm.hash` |
