---
title: Local development
---

# Local development

`nwm dev` turns a recipe into a development environment for your own packages. It installs a server, links your package
folders into it, runs your build tools and reloads a package in the running server as soon as you save a file.

```bash
cd my-gamemode
nwm dev
```

Anyone who clones your repository gets the same environment with the same command. Everyone else can still install
the recipe as a normal server, see [One recipe, two modes](#one-recipe-two-modes).

## A minimal setup

Put a `Recipefile` next to your package:

```text
my-gamemode/
├── Recipefile
├── Package.toml
├── Server/
│   └── Index.lua
└── Client/
    └── Index.lua
```

```recipefile
recipe @you/my-gamemode 0.1.0

package my-gamemode path . enable
package sandbox

dev package test-utils path tests/utils enable
dev config debug.log_level 2
```

Run `nwm dev`. Edit `Server/Index.lua` and the server prints:

```text
[nwm] ♻️  hot reloading my-gamemode (1 changes)
INFO  CONSOLE: 'package hotreload my-gamemode'.
INFO  Package 'my-gamemode' hot reloaded.
```

## What nwm dev does

1. Finds or creates the development server and remembers it (see [Where the server lives](#where-the-server-lives))
2. Applies the recipe including all `dev` statements: vault packages, services, secrets, config. When the recipe did
   not change since the last run, this step is skipped and only the local packages are linked again
3. Builds packages that need a build step once
4. **Mounts** local packages into the server (see [How packages are mounted](#how-packages-are-mounted))
5. Starts watch builders such as `nebra watch` in the background
6. Starts the server with its console attached to your terminal
7. Watches your files and reloads changed packages

Press `Ctrl+C` or type `/quit` to stop everything. The server is stopped gracefully through its console.

## Reloading

Changes are collected for 300 ms and then applied in one go, so saving many files at once causes a single reload.

| What changed | What nwm does |
|---|---|
| Lua or other files of a package | `package hotreload <name>`: the new code runs, the Lua state is kept |
| A package with `reload full` | `package reload <name>`: the package is unloaded and loaded fresh |
| `Package.toml` | Copies it again and runs `package reload <name>` |
| A new top-level folder, e.g. `Shared/` | Mounts it and reloads the package |
| Sources of a package with `build` but no `watch` | Runs the build, then reloads |
| An asset pack | Asks you to `/restart`, asset packs cannot be reloaded at runtime |

Pick the reload mode per package:

```recipefile
package my-gamemode path . enable reload full
```

Use `hot` (the default) while iterating on logic, `full` when your package sets up state on load that a hot reload
would duplicate, for example entities spawned at startup.

## The console

Everything you type goes to the server console, so all
[server console commands](https://docs.nanos-world.com) work. Lines starting with `/` are nwm commands:

| Command | Effect |
|---|---|
| `/hotreload [package...]` | Hot reload packages, all local packages without arguments |
| `/reload [package...]` | Fully reload packages |
| `/restart` | Restart the server, builders and watchers keep running |
| `/stop`, `/start` | Stop or start the server |
| `/status` | Show the mounted packages and how they are mounted |
| `/quit` | Stop everything |

From another terminal you can reach the same server with `nwm server cmd`:

```bash
nwm server cmd -s my-gamemode-dev package reload all
```

## Build tools and Nebra

Packages that need a build step get it automatically when nwm recognises the project:

| Detected | Build | Watch | Output |
|---|---|---|---|
| `nebra.toml` in the package folder | `nebra build` | `nebra watch` | `output` from `nebra.toml` (default `out`) |

The `source` and `output` folders and `nebra.toml` itself are not part of the mounted package. The compiled files from the
output folder are, merged with everything else in the package folder (for example `Package.toml` or UI files in
`Client/`). Declare the tool so people without it get a helpful message:

```recipefile
tool nebra
package my-gamemode path . enable
```

For other tools, say how to build:

```recipefile
package ui path ui enable build "npm run build" watch "npm run watch" output dist
package lib path lib enable build "make lua" output out
```

| Option | Meaning |
|---|---|
| `build "<command>"` | Runs once before the package is installed, and in `nwm dev` on every source change when there is no `watch` |
| `watch "<command>"` | A long running builder started by `nwm dev`, its output is shown with a prefix |
| `output <dir>` | Folder with the built files, merged into the package |

Commands run directly, without a shell, so they behave the same on Linux and Windows. Explicit options override what
nwm detects.

## How packages are mounted

nwm does not link your whole package folder into the server. It builds the package from pieces:

```text
Packages/my-gamemode/
├── Package.toml     a copy, see below
├── Server/   → link to my-gamemode/Server (or the build output)
├── Client/   → link to my-gamemode/Client
└── Shared/   → link to my-gamemode/Shared
```

- **`Package.toml` is copied**, because the server rewrites it on load (it adds missing keys and comments). Your file in
  git stays untouched; nwm copies it again when you change it.
- **Folders are linked**: symlinks on Linux, directory junctions on Windows. Junctions need neither administrator rights
  nor developer mode.
- **Folders that exist in the package and in the build output** are merged into a real folder and synchronised on
  changes.
- **Left out**: hidden files and folders, recipe files, build sources and outputs, `nebra.toml`, and folders of other
  local packages nested inside this one.

When links cannot be created, for example on network drives, exFAT or `\\wsl$` paths, nwm copies the folders instead and
keeps them in sync. `/status` shows which mode each package uses.

Removing a server with `nwm server remove --delete-files` removes links without following them, so your sources are
never deleted.

## Several packages and repositories

List every package you work on. Packages in other repositories can be developed side by side with `local`:

```recipefile
package my-gamemode path . enable
package my-ui git https://github.com/you/my-ui.git ref v1.0.0 local ../my-ui enable
```

In `nwm dev`, if `../my-ui` exists it is linked and watched like any local package. If it does not, for example for a
teammate who only works on the game mode, nwm installs `v1.0.0` from git. `local` also works for vault packages, so you
can develop a fork of a published package.

## One recipe, two modes

The same recipe serves development and normal installs:

| | `nwm dev` | `nwm server create --recipe` / `nwm recipe apply` |
|---|---|---|
| `path` and `local` packages | Linked, watched, hot reloaded | Built once and copied |
| `dev` statements | Applied | Ignored |
| `watch` commands | Running | Not used |

So you can publish the recipe of your game mode and others install it as a regular server, while your own repository
doubles as its development environment. Build commands of remote recipes need confirmation like hooks, or `--yes`.

## Dev only statements

Prefix any statement with `dev` to use it only in `nwm dev`:

```recipefile
dev package test-utils path tests/utils enable
dev package admin-tools enable
dev config debug.log_level 3
dev arg --log_level 3
dev use redis
dev env DEBUG true
```

`recipe`, `description`, `author` and `keywords` cannot be dev only.

## Where the server lives

| Command | Server folder |
|---|---|
| `nwm dev` | `~/.nwm/servers/<recipe>-dev`, e.g. `my-gamemode-dev` |
| `nwm dev --local` | `.nwm/server` next to the recipe, with a `.gitignore` that keeps it out of git |
| `nwm dev -d <dir>` | `<dir>` |
| `nwm dev --name <name>` | Use another server name |

The development server is a regular server, so `nwm server info`, `nwm server logs`, `nwm service list` and friends work
with it.

## Flags

```text
nwm dev [directory | file] [-- server arguments] [flags]
```

| Flag | Effect |
|---|---|
| `-n, --name <name>` | Name of the development server |
| `-d, --dir <dir>` | Folder of the development server |
| `--local` | Keep the server in `.nwm/` next to the recipe |
| `--reapply` | Apply the whole recipe even if it did not change |
| `--allow-unsafe-libs` | Approve `--enable_unsafe_libs` without asking |
| `--var`, `--secret` | Recipe variables and secrets |
| `--no-services`, `--no-hooks` | Skip services or hooks |

Arguments after `--` are passed to the server, e.g. `nwm dev -- --log_level 3`.
