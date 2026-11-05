---
title: Applying recipes
---

# Applying recipes

## Create a server from a recipe

```bash
nwm server create <name> --recipe <source> [flags]
```

nwm fetches and validates the recipe first, resolves variables and secrets, then installs the server (using the
recipe's `branch` unless you pass `--branch`) and applies the recipe. If applying fails after the server was installed,
fix the problem and run `nwm recipe apply` instead of creating it again.

## Apply to an existing server

```bash
nwm recipe apply <source> [-s server] [flags]
```

The server has to be stopped. Applying again is safe: packages are reinstalled or updated, config values are merged
and services are recreated only when their definition changed.

## Upgrade

```bash
nwm recipe upgrade [-s server] [flags]
```

Applies the server's recipe again from the source it was created with and picks up the newest version that matches its
[version constraint](./sources.md#versions). Variables from the original apply are reused, `--var` changes them. If the
version and content did not change, nwm says so and does nothing.

## Flags

These flags work for `server create --recipe`, `recipe apply` and `recipe upgrade`:

| Flag | Effect |
|---|---|
| `--var name=value` | Set a [variable](./variables.md), repeatable |
| `--secret name=value` | Set a [secret](../secrets.md) instead of generating or asking, repeatable |
| `--no-services` | Do not start the recipe's services now (they still start with the server) |
| `--no-hooks` | Do not run hooks |
| `-y, --yes` | Run hooks of remote recipes without asking |
| `--refresh` | Download the registry index again |

## What happens

nwm applies a recipe in this order, showing a progress line per step:

1. **Secrets**: generate missing `random` secrets, ask for `prompt` secrets
2. **Vault packages and asset packs** through the server's own CLI
3. **Git and recipe packages and asset packs**
4. **C modules**
5. **Files**, templates are rendered
6. **Configuration**: merge `config` values into `Config.toml`, enable resources, write `.env`
7. **NanosSharp**, if requested and not installed yet
8. **Record** the recipe, launch arguments and services in the server's `nwm.toml`
9. **Services** are started and nwm waits until they are ready
10. **Hooks** run, after confirmation for remote recipes

## What gets recorded

The server's `nwm.toml` stores what was applied, so later commands know about it:

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
```

`nwm server start` reads `args` and starts the services. `nwm recipe upgrade` reads `recipe`.
