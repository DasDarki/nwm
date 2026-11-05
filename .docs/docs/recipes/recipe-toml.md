---
title: recipe.toml
---

# recipe.toml

`recipe.toml` holds exactly the same data as a [`Recipefile`](./recipefile.md), written as TOML. Use whichever you
prefer; `nwm recipe convert` switches between them and `nwm recipe show --format toml` prints any recipe as TOML.

nwm rejects unknown keys in `recipe.toml`, so typos are reported instead of ignored.

## Example

This `Recipefile`:

```recipefile
recipe @you/roleplay 1.0.0
description "Roleplay development server"
author you
keywords roleplay postgres

branch public
nanossharp
arg --log_level 2

var port 7777

config discover.name "RP ${server.name}"
config discover.port ${port}

package sandbox enable
package rp-core enable git https://github.com/you/rp-core.git ref v2.1.0
package dev-tools path packages/dev-tools

module mysql linux https://example.com/libmysql.so windows https://example.com/mysql.dll enable

file files/NanosSharp.toml NanosSharp.toml template

use postgres

secret steam_token prompt
env STEAM_TOKEN ${secret.steam_token}

hook in Packages/rp-core dotnet build
```

is this `recipe.toml` (generated with `nwm recipe show --format toml`):

```toml
[recipe]
name = '@you/roleplay'
version = '1.0.0'
description = 'Roleplay development server'
authors = ['you']
keywords = ['roleplay', 'postgres']

[server]
branch = 'public'
nanossharp = true
args = ['--log_level', '2']

[variables]
port = 7777

[config]
[config.discover]
name = 'RP ${server.name}'
port = '${port}'

[env]
DATABASE_URL = 'postgres://nanos:${secret.postgres_password}@127.0.0.1:5432/nanos'
STEAM_TOKEN = '${secret.steam_token}'

[[package]]
name = 'sandbox'
enable = true

[[package]]
name = 'rp-core'
git = 'https://github.com/you/rp-core.git'
ref = 'v2.1.0'
enable = true

[[package]]
name = 'dev-tools'
path = 'packages/dev-tools'

[[module]]
name = 'mysql'
linux = 'https://example.com/libmysql.so'
windows = 'https://example.com/mysql.dll'
enable = true

[[file]]
source = 'files/NanosSharp.toml'
target = 'NanosSharp.toml'
template = true

[[service]]
name = 'postgres'
image = 'postgres:17'
ports = ['5432']
volumes = ['postgres-data:/var/lib/postgresql/data']
ready = ['pg_isready', '-h', '127.0.0.1', '-U', 'nanos', '-d', 'nanos']

[service.env]
POSTGRES_DB = 'nanos'
POSTGRES_PASSWORD = '${secret.postgres_password}'
POSTGRES_USER = 'nanos'

[[secret]]
name = 'postgres_password'

[[secret]]
name = 'steam_token'
prompt = true

[[hook]]
run = ['dotnet', 'build']
dir = 'Packages/rp-core'
```

## Tables

| Table | Recipefile statements |
|---|---|
| `[recipe]` with `name`, `version`, `description`, `authors`, `keywords` | `recipe`, `description`, `author`, `keywords` |
| `[server]` with `branch`, `nanossharp`, `args` | `branch`, `nanossharp`, `arg` |
| `[variables]` | `var` |
| `[config]` and its sub tables | `config` |
| `[env]` | `env` |
| `[[package]]`, `[[assets]]` with `name`, `git`, `ref`, `subdir`, `path`, `enable` | `package`, `assets` |
| `[[module]]` with `name`, `linux`, `windows`, `enable` | `module` |
| `[[file]]` with `source`, `target`, `template` | `file` |
| `[[service]]` with `name`, `image`, `ports`, `env`, `volumes`, `command`, `ready`, `wait` | `service` (and `use`) |
| `[[secret]]` with `name`, `length`, `prompt` | `secret` |
| `[[hook]]` with `run`, `dir` | `hook` |

`use` has no table of its own: it is stored as the `[[service]]`, `[[secret]]` and `[env]` entries it expands to.
Values in `[service.env]` must be strings.
