---
title: Variables and placeholders
sidebar_label: Variables
---

# Variables and placeholders

Recipes use `${...}` placeholders for values that differ between servers or should not be written down.

## Recipe variables

```recipefile
var port 7777
var region eu

config discover.port ${port}
config discover.name "Roleplay ${region}"
```

Override them when applying:

```bash
nwm server create rp --recipe @you/roleplay --var port=7790 --var region=us
```

Values given with `--var` are typed like values in the recipe: `--var port=7790` is a number, `--var debug=true` a
boolean. nwm stores the variables you passed in the server's `nwm.toml`, so `nwm recipe upgrade` reuses them. Passing
`--var` again on upgrade changes them.

## Built-in variables

| Placeholder | Value |
|---|---|
| `${server.name}` | Name of the server the recipe is applied to |
| `${server.path}` | Absolute path of the server folder |
| `${os}` | `linux` or `windows` |

## Secrets

`${secret.name}` inserts a value from the server's [secret store](../secrets.md). The secret has to be declared with
`secret name` in the recipe or set with `nwm secret set`.

## Other env values

`${env.NAME}` inserts a value of an [`env`](./recipefile.md#env) statement of the recipe, including the ones
[`use`](./recipefile.md#use) creates. The order of the lines does not matter, and circular references are reported as an
error.

```recipefile
use postgres port 15432
env DATABASE_URL_NOSSL ${env.DATABASE_URL}?sslmode=disable
config custom_settings.database_url ${env.DATABASE_URL}
```

`${env.NAME}` only sees values from the recipe, not the environment of your shell or keys added to `.env` by hand. Env
values are always strings, use [`var`](./recipefile.md#var) for numbers that have to stay numbers in `Config.toml`.

## Where placeholders work

| Place | Notes |
|---|---|
| `config` values | A value that is only a placeholder keeps its type: `config discover.port ${port}` stays a number |
| `env` values | Always strings |
| `arg` | Launch arguments |
| `package` / `assets` | `git` and `ref` |
| `service` / `use` | Image, ports, volumes, env, command and ready check. Secrets in service env are resolved only when the container starts, so `nwm.toml` never contains passwords |
| `hook` | Command and directory |
| `file ... template` | Anywhere in the file content |

Unknown placeholders are an error that lists the missing names, so typos do not end up in your config.
`${env.NAME}` inside a service definition keeps secrets as placeholders too, so `nwm.toml` stays free of passwords.
