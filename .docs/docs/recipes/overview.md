---
title: Overview
---

# Recipes

A recipe describes a complete server setup as code. Instead of a wiki page that says "install these packages, set these
config values, start a database, copy this file", you write it down once and let nwm do it:

```recipefile
recipe @you/roleplay 1.0.0
description "Roleplay development server"

nanossharp
var port 7777

config discover.name "RP ${server.name}"
config discover.port ${port}

package sandbox enable
package rp-core git https://github.com/you/rp-core.git ref v2.1.0 enable

use postgres
use redis

secret steam_token prompt
env STEAM_TOKEN ${secret.steam_token}
hook in Packages/rp-core dotnet build
```

```bash
nwm server create rp --recipe ./roleplay
```

## What a recipe can contain

| Part | Example | Details |
|---|---|---|
| Metadata | `recipe @you/roleplay 1.0.0` | [Recipefile](./recipefile.md#recipe) |
| Server settings | `branch`, `nanossharp`, `arg` | [Recipefile](./recipefile.md#branch) |
| Variables | `var port 7777` | [Variables](./variables.md) |
| Config.toml values | `config discover.port ${port}` | [Recipefile](./recipefile.md#config) |
| Packages and asset packs | `package sandbox enable` | [Recipefile](./recipefile.md#package) |
| C modules | `module mysql linux https://...` | [C modules](../servers/c-modules.md) |
| Files | `file files/NanosSharp.toml NanosSharp.toml template` | [Recipefile](./recipefile.md#file) |
| Services | `use postgres` | [Services](../services.md) |
| Secrets | `secret steam_token prompt` | [Secrets](../secrets.md) |
| Environment | `env STEAM_TOKEN ${secret.steam_token}` | [Recipefile](./recipefile.md#env) |
| Hooks | `hook dotnet build` | [Recipefile](./recipefile.md#hook) |

## Two formats

Recipes are usually written as a **`Recipefile`**, a line based format designed for humans and for the
[interactive editor](./editing.md). nwm also understands **`recipe.toml`**, which holds the same data as TOML tables.
Convert between them at any time with `nwm recipe convert --to toml` or `--to dsl`. See
[recipe.toml](./recipe-toml.md).

A folder can also hold several recipes with their own names, such as `dev.Recipefile` and `staging.Recipefile`, which
share the same `files/` and `packages/` folders.

## Lifecycle

1. **Write** it with `nwm recipe init`, `nwm recipe edit` or `nwm recipe add`, see [Editing recipes](./editing.md)
2. **Check** it with `nwm recipe validate`
3. **Apply** it with `nwm server create --recipe` or `nwm recipe apply`, see [Applying recipes](./applying.md)
4. **Share** it through git, a URL, a gist or the [registry](../registry/publishing.md)
5. **Upgrade** servers with `nwm recipe upgrade` when the recipe changes
