---
title: Quick start
---

# Quick start

This page takes you from zero to a running server in a few minutes. It assumes nwm is
[installed](./installation.mdx).

## 1. Create a server

```bash
nwm server create dev
```

nwm downloads SteamCMD if needed, installs the nanos world dedicated server into `~/.nwm/servers/dev` and registers
it under the name `dev`. Want a different folder or port?

```bash
nwm server create dev --dir ./server --port 7777 --display-name "My dev server"
```

## 2. Add a game mode

Install the sandbox game mode from the vault and enable it in `Config.toml`:

```bash
nwm package install sandbox --enable
```

`--enable` knows that `sandbox` is a game mode and sets `game.game_mode`. Script packages and C modules are added to
`game.packages` instead.

## 3. Start it

```bash
nwm server start
```

The server runs in your terminal until you press `Ctrl+C`. To run it in the background:

```bash
nwm server start -d
nwm server logs -f
nwm server stop
```

:::tip You rarely need to type the server name
When you leave out the name, nwm picks the server whose folder you are in, or the only server you have.
Most commands also have short aliases: `nwm s start -d` is the same as `nwm server start --detach`.
:::

## 4. Describe it as a recipe

Once your server grows, put its setup into a recipe so you can rebuild or share it:

```bash
mkdir my-server && cd my-server
nwm recipe init --name @you/my-server
nwm recipe add package sandbox enable
nwm recipe add use postgres
nwm recipe edit
```

`nwm recipe edit` opens an interactive editor with tab completion for commands, store packages, service presets and
config keys. Then build a fresh server from it:

```bash
nwm server create staging --recipe .
nwm server start staging
```

This installs the server, the sandbox package and a Postgres container with a generated password, and writes
`DATABASE_URL` to the server's `.env`.

## Where to go next

- [Core concepts](./concepts.md) explains servers, recipes, services and where nwm stores things
- [Recipefile reference](../recipes/recipefile.md) lists every statement
- [CLI reference](../reference/cli.md) lists every command and flag
