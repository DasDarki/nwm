---
slug: /intro
title: What is nwm?
sidebar_label: Introduction
---

# What is nwm?

**nwm**, the *nanos world manager*, is a command line tool for Linux and Windows that takes the busywork out of
running [nanos world](https://nanos-world.com) servers. It installs and updates servers through SteamCMD, starts and
stops them, manages packages and asset packs from the vault, installs [NanosSharp](https://nanossharp.dasdarki.de)
and runs databases and other services next to your server.

The core idea is the **recipe**: a single `Recipefile` that describes a complete server. Check it into git, share it as a
gist or publish it to the registry, and anyone can recreate the same server with one command:

```bash
nwm server create rp --recipe @dasdarki/roleplay
```

## What nwm does for you

| Area | What you get |
|---|---|
| [Servers](./servers/creating.md) | Install, update and run servers, switch Steam branches, follow logs |
| [Configuration](./servers/configuration.md) | Read and change `Config.toml` without breaking it |
| [Packages and assets](./servers/packages-and-assets.md) | Install, update and enable resources from the vault |
| [NanosSharp](./servers/nanossharp.md) | Add C# scripting to a server in one step |
| [C modules](./servers/c-modules.md) | Ship native modules as packages, with a safety prompt for unsafe libraries |
| [Recipes](./recipes/overview.md) | Describe a whole server as code, with variables, files and hooks |
| [Services](./services.md) | Postgres, MySQL, Redis and friends as Docker containers, wired into `.env` |
| [Secrets](./secrets.md) | Generated or prompted passwords, kept out of recipes and server folders |
| [Registry](./registry/using.md) | Publish and find recipes as `@scope/name`, versioned with git tags |

## How the pieces fit together

```text
             Recipefile ──────────────┐
  (folder, git repo, URL, gist,       │  nwm server create --recipe
   or @scope/name from the registry)  ▼
                               ┌───────────────┐
  SteamCMD ──── install ─────▶ │  your server  │ ◀── nwm package / assets
                               │  Config.toml  │
  ~/.nwm/secrets ── values ──▶ │  .env         │ ◀── nwm secret
                               │  nwm.toml     │
                               └───────┬───────┘
                                       │ nwm server start
                                       ▼
                       Docker services (postgres, redis, ...)
                       then the nanos world server itself
```

## Next steps

- [Install nwm](./getting-started/installation.mdx)
- [Create your first server](./getting-started/quick-start.md)
- [Learn the core concepts](./getting-started/concepts.md)

:::note
nwm is a community project and is not affiliated with the nanos world team.
:::
