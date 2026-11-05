---
title: Recipefile reference
sidebar_label: Recipefile reference
---

# Recipefile reference

A `Recipefile` is a plain text file with one statement per line. Every statement starts with a command, followed by
arguments and options.

```recipefile
# comments start with #
package sandbox enable
use redis port 16379
```

## Syntax

| Rule | Example |
|---|---|
| One statement per line, `#` starts a comment | `# database` |
| Words are separated by spaces | `package sandbox enable` |
| `"..."` groups words and supports escapes (`\"`, `\\`, `\n`) | `description "My \"best\" server"` |
| `'...'` groups words literally | `env PATTERN 'C:\paths\as\is'` |
| `[...]` and `{...}` are read as one value, such as a list | `config game.assets [nanos-world, cyberpunk]` |
| Options can come in any order after the positional arguments | `package x enable git https://...` |
| `${name}` is replaced by a variable, built-in or secret | `config discover.port ${port}` |

### Value types

For `var` and `config` the type of a value matters, because nanos world expects numbers and booleans in
`Config.toml`:

| Written as | Becomes |
|---|---|
| `7777` | number |
| `1.5` | number |
| `true`, `false` | boolean |
| `[a, b]`, `["a", "b"]` | list |
| `"7777"`, `'true'` | string, because it is quoted |
| `sandbox` | string |
| `${port}` | the type of the variable (here a number) |

### Repeating statements

Named entries replace an earlier entry with the same name: `package`, `assets`, `module`, `service`, `use`, `secret`,
`var`, `config`, `env` and `file` (by target). `arg`, `hook`, `author` and `keywords` add to the list instead.
The order of statements does not matter: nwm always applies a recipe in a [fixed order](./applying.md#what-happens).

### Short aliases

| Statement | Aliases |
|---|---|
| `package` | `pkg` |
| `assets` | `asset` |
| `module` | `mod` |
| `config` | `cfg` |
| `service` | `svc` |
| `use` | `require_service`, `requires`, `require` |

When nwm writes a `Recipefile` (from the editor, `add` or `convert`) it uses the full names.

---

## recipe

```recipefile
recipe <name> [version]
```

The name and version of the recipe. Names are lowercase and may be scoped like `@you/name`; a scope is required to
[publish](../registry/publishing.md) to the registry. The version is informational, released versions are git tags.

```recipefile
recipe @you/roleplay 1.0.0
```

## description

```recipefile
description <text>
```

A short description, shown in `nwm recipe info` and the registry.

## author

```recipefile
author <name>...
```

Adds one or more authors.

## keywords

```recipefile
keywords <word>...
```

Adds search keywords for `nwm recipe search`.

## branch

```recipefile
branch public|bleeding-edge
```

The Steam branch used when a server is created from the recipe. `nwm server create --branch` overrides it.

## nanossharp

```recipefile
nanossharp [on|off]
```

Installs [NanosSharp](../servers/nanossharp.md) into the server. Without an argument it is `on`.

## arg

```recipefile
arg <argument>...
```

Launch arguments for the server, stored in the server's `nwm.toml` and added on every `nwm server start`.

```recipefile
arg --log_level 2
arg --max_tick_rate 60
```

`arg --enable_unsafe_libs` is allowed but needs your approval on the first start, see
[C modules](../servers/c-modules.md#unsafe-libraries).

## var

```recipefile
var <name> <default>
```

Defines a variable with a default value, usable as `${name}` anywhere in the recipe and overridable with
`--var name=value`. See [Variables](./variables.md).

```recipefile
var port 7777
var motd "Welcome to ${server.name}"
```

## config

Alias: `cfg`

```recipefile
config <key> <value>
```

Sets a value in the server's `Config.toml`. Keys use dots for sections. Values are typed as described
[above](#value-types). Keys under `custom_settings.` are free, other unknown keys produce a warning because they are
usually typos.

```recipefile
config discover.name "RP ${server.name}"
config discover.port ${port}
config general.max_players 32
config custom_settings.spawn_cars true
```

## package

Alias: `pkg`

```recipefile
package <name> [enable] [git <url>] [ref <ref>] [subdir <dir>] [path <dir>]
```

Installs a package into `Packages/<name>`.

| Source | Example |
|---|---|
| Vault (default) | `package sandbox` |
| Git repository | `package rp-core git https://github.com/you/rp-core.git ref v2.1.0` |
| Folder in a git repository | `package ui git https://github.com/you/monorepo.git subdir packages/ui` |
| Folder in the recipe | `package dev-tools path packages/dev-tools` |

| Option | Effect |
|---|---|
| `enable` | Load the package on start, see [where it goes](../servers/packages-and-assets.md#enable-and-disable) |
| `git <url>` | Clone the package from this repository (needs `git`) |
| `ref <ref>` | Tag or branch to clone, default branch otherwise |
| `subdir <dir>` | Use a folder of the repository as the package |
| `path <dir>` | Copy the package from the recipe folder |

Git and recipe packages replace an existing folder of the same name. The `.git` folder is not copied.

## assets

Alias: `asset`

```recipefile
assets <name> [enable] [git <url>] [ref <ref>] [subdir <dir>] [path <dir>]
```

Same as [`package`](#package) for asset packs in `Assets/`. `enable` adds the pack to `game.assets`.

## module

Alias: `mod`

```recipefile
module <name> [linux <source>] [windows <source>] [enable]
```

Builds a [C module](../servers/c-modules.md) package from prebuilt libraries. A source is a URL or a path in the recipe
and may point to a `.so`/`.dll` or a `.zip`/`.tar.gz` archive. Only the source for the current platform is installed;
a platform without a source is skipped.

```recipefile
module mysql linux https://example.com/libmysql.so windows https://example.com/mysql.dll enable
```

## file

```recipefile
file <source> <target> [template]
```

Copies a file or folder from the recipe into the server. The target is relative to the server folder and cannot leave
it. With `template` nwm replaces `${...}` placeholders in the file, including secrets.

```recipefile
file files/NanosSharp.toml NanosSharp.toml
file files/settings.json Packages/rp-core/settings.json template
```

## use

Aliases: `require_service`, `requires`, `require`

```recipefile
use <image> [as <name>] [port <host port>] [env KEY=value]...
```

Adds a [Docker service](../services.md) with sensible defaults. For known images (`postgres`, `mysql`, `mariadb`,
`redis`, `valkey`, `mongo`, `rabbitmq`, `meilisearch`, `minio`) nwm picks the image tag, port, data volume and a
readiness check, generates a password [secret](../secrets.md) and writes connection values such as `DATABASE_URL` to
`.env`. See the [preset table](../services.md#presets).

| Option | Effect |
|---|---|
| `as <name>` | Service name, defaults to the image name |
| `port <host port>` | Publish on another host port, connection URLs follow |
| `env KEY=value` | Extra container environment variable, repeatable |

```recipefile
use postgres
use redis port 16379
use mysql as legacy-db port 13306
```

`use` is shorthand: it is stored as a `service`, `secret` and `env` statements, which is what you see after saving the
recipe from the editor.

## service

Alias: `svc`

```recipefile
service <name> <image> [port <p>]... [env KEY=value]... [volume <v>]... [ready "<check>"] [nowait] [command <args>...]
```

Adds a Docker container that starts with the server.

| Option | Effect |
|---|---|
| `port <p>` | Publish a port: `5432`, `15432:5432` or `0.0.0.0:5432:5432`. Without an IP it binds to `127.0.0.1` |
| `env KEY=value` | Container environment variable. Values may use `${secret.name}` |
| `volume <v>` | `name:/path` for a volume managed by nwm, `./dir:/path` for a folder relative to the server |
| `ready "<check>"` | Command run inside the container until it succeeds, e.g. `ready "pg_isready -U app"` |
| `nowait` | Do not wait for the service to become ready |
| `command <args>...` | Override the container command. Takes the rest of the line, so put it last |

```recipefile
service search getmeili/meilisearch:v1 port 7700 env MEILI_ENV=development volume search-data:/meili_data
```

## secret

```recipefile
secret <name> [random [length]] | [prompt]
```

Declares a [secret](../secrets.md), used as `${secret.name}`.

| Form | Effect |
|---|---|
| `secret db_password` | Random value with 24 characters, generated on the first apply |
| `secret api_key random 48` | Random value with 48 characters (8 to 256) |
| `secret steam_token prompt` | Asked for when the recipe is applied, or passed with `--secret steam_token=...` |

Existing values are kept, so upgrades do not change passwords.

## env

```recipefile
env <KEY> <value>
```

Writes a variable to the server's `.env` file. nwm passes these variables to the server process on every start.
Existing keys in `.env` that the recipe does not mention are kept.

```recipefile
env API_URL https://api.example.com
env STEAM_TOKEN ${secret.steam_token}
```

## hook

```recipefile
hook [in <dir>] <command>...
```

Runs a command after the recipe was applied, in the server folder or in `<dir>` relative to it. Hooks see the
variables `NWM_SERVER`, `NWM_SERVER_DIR` and `NWM_RECIPE_DIR`.

```recipefile
hook in Packages/rp-core dotnet build
hook sh -c "echo applied to $NWM_SERVER"
```

Hooks of remote recipes (git, URL, gist, registry) are shown and need confirmation unless you pass `--yes`. Without a
terminal they are skipped. `--no-hooks` skips them always.
