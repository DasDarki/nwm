---
title: Core concepts
---

# Core concepts

## Servers

A **server** is a nanos world dedicated server installation that nwm knows by name. You either create it with
`nwm server create` (nwm installs it through SteamCMD) or register an existing installation with `nwm server import`.
nwm remembers each server's name, folder and Steam branch in `~/.nwm/state.toml`.

### Which server a command uses

Commands that work on a server accept its name as an argument (`nwm server start dev`) or through `-s/--server`
(`nwm package install sandbox -s dev`). Without a name nwm picks, in this order:

1. the server whose folder contains your current directory,
2. the only registered server, if there is exactly one.

Otherwise it asks you to pass a name.

## Recipes

A **recipe** describes everything a server needs: its Steam branch, launch arguments, `Config.toml` values, packages,
asset packs, C modules, files, `.env` values, Docker services, secrets and commands to run afterwards. Recipes are
written as a [`Recipefile`](../recipes/recipefile.md) (or [`recipe.toml`](../recipes/recipe-toml.md)) and can come from a
folder, a git repository, a URL, a gist or the [registry](../registry/using.md).

Applying a recipe is repeatable: `nwm recipe upgrade` applies it again and picks up new versions.

## Services

A **service** is a Docker container that belongs to a server, for example a database. Services are listed in the
server's `nwm.toml`, start before the server and are reachable on `127.0.0.1`. See [Services](../services.md).

## Secrets

**Secrets** are passwords and tokens stored per server in `~/.nwm/secrets/<server>.toml`, outside of recipes and server
folders. Recipes reference them as `${secret.name}`. See [Secrets](../secrets.md).

## Where things live

| Path | Content |
|---|---|
| `~/.nwm/state.toml` | Registered servers, approvals for unsafe libraries |
| `~/.nwm/servers/<name>/` | Default location of servers created by nwm |
| `~/.nwm/secrets/<name>.toml` | Secrets of a server, readable only by you |
| `~/.nwm/logs/<name>.log` | Output of servers started with `--detach` |
| `~/.nwm/steamcmd/` | SteamCMD downloaded by nwm |
| `~/.nwm/cache/` | Registry index, store catalog, recipe checkouts, downloads |

Inside each server folder nwm uses:

| File | Content |
|---|---|
| `Config.toml` | The server configuration, see [Configuration](../servers/configuration.md) |
| `nwm.toml` | Applied recipe, launch arguments and services |
| `.env` | Environment variables passed to the server process |
| `Packages/`, `Assets/` | Installed packages and asset packs |

Set `NWM_HOME` to move `~/.nwm` somewhere else. All paths are listed in the [files reference](../reference/files.md).
