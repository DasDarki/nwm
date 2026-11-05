---
title: Packages and assets
---

# Packages and asset packs

nwm installs resources from the nanos world vault by calling the server's own command line interface
(`NanosWorldServer --cli install package ...`), so dependencies are resolved exactly like in the game.
`nwm package` (alias `nwm p`) handles packages, `nwm assets` (alias `nwm a`) asset packs. Both have the same commands.

```bash
nwm package install <name>... [--enable] [-s server]
nwm package update [name]... [-s server]
nwm package list [-s server]
nwm package enable <name>... [-s server]
nwm package disable <name>... [-s server]
nwm package remove <name>... [-s server]
```

## Install

```bash
nwm package install sandbox battlefield-kill-ui --enable
nwm assets install nanos-world
```

After the vault finished, nwm checks that every requested resource actually arrived in `Packages/` or `Assets/` and
reports the ones that did not.

## Update

```bash
nwm package update
nwm package update sandbox
```

Without names nwm updates every installed package (or asset pack).

## Enable and disable

Installing a resource does not load it. Enabling writes it to `Config.toml`:

| Package type | Where it goes |
|---|---|
| `game_mode` | `game.game_mode` (replaces the current one) |
| `map` | `game.map` |
| `loading_screen` | `game.loading_screen` |
| `script`, `c_module` | added to `game.packages` |
| asset packs | added to `game.assets` |

`disable` removes the resource from wherever it is set. `remove` disables it and deletes its folder.

## List

`nwm package list` shows every installed package with its title, type, version, author and whether it is enabled.

## Packages from git or your own folders

The vault is not the only source. [Recipes](../recipes/recipefile.md#package) can install packages from git repositories
(`package rp-core git https://github.com/you/rp-core.git ref v2.1.0`) or from a folder inside the recipe
(`package dev-tools path packages/dev-tools`).
