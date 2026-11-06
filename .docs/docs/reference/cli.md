---
title: CLI reference
---

# CLI reference

Every command, its aliases and its flags. Run `nwm <command> --help` for the same information in your terminal.

Commands that work on a server take its name as an argument or through `-s/--server`. Without a name nwm uses the
server whose folder you are in, or the only server you have.

## Aliases

| Command | Aliases |
|---|---|
| `nwm server` | `s`, `servers`, `srv` |
| `nwm package` | `p`, `packages`, `pkg` |
| `nwm assets` | `a`, `asset` |
| `nwm nanossharp` | `n`, `ns`, `sharp`, `cs` |
| `nwm recipe` | `r`, `recipes` |
| `nwm service` | `svc`, `services` |
| `nwm secret` | `sec`, `secrets` |
| `nwm self-update` | `su`, `upgrade` |

## nwm server

### server create

```text
nwm server create <name> [flags]
```

| Flag | Description |
|---|---|
| `-b, --branch <branch>` | Steam branch: `public` or `bleeding-edge` (default `public`, or the recipe's branch) |
| `-d, --dir <path>` | Installation folder (default `~/.nwm/servers/<name>`) |
| `--display-name <name>` | Server name shown in the server list |
| `--force` | Install into a folder that is not empty |
| `--game-mode <package>` | Game mode package |
| `--map <package>` | Map package |
| `--max-players <n>` | Maximum players |
| `--nanossharp` | Install NanosSharp right after the server |
| `--port <port>` | Game port |
| `--query-port <port>` | Query port |
| `-r, --recipe <source>` | Set the server up from a [recipe](../recipes/sources.md) |
| `--var <name=value>` | Recipe variable, repeatable |
| `--secret <name=value>` | Recipe secret, repeatable |
| `--no-services` | Do not start the recipe's services |
| `--no-hooks` | Do not run the recipe's hooks |
| `-y, --yes` | Run hooks of remote recipes without asking |
| `--refresh` | Refresh the registry index first |

### server import

```text
nwm server import <name> <directory> [-b public|bleeding-edge]
```

Registers an existing installation.

### server update

```text
nwm server update [name] [-b branch] [--no-validate]
```

Installs the newest build of the server's branch, or switches branch with `-b`.

### server list, info

```text
nwm server list
nwm server info [name]
```

Alias of `list`: `ls`.

### server start

```text
nwm server start [name] [-- server arguments] [flags]
```

| Flag | Description |
|---|---|
| `-d, --detach` | Run in the background and write output to the log file |
| `--no-services` | Do not start the server's Docker services |
| `--allow-unsafe-libs` | Approve `--enable_unsafe_libs` from the recipe without asking |

### server stop

```text
nwm server stop [name] [--timeout 15s] [--services]
```

| Flag | Description |
|---|---|
| `--timeout <duration>` | Time to wait before killing the server (default `15s`) |
| `--services` | Also stop the server's Docker services |

### server restart

```text
nwm server restart [name] [-- server arguments] [--timeout 15s] [--allow-unsafe-libs]
```

Stops the server if needed and starts it in the background.

### server logs

```text
nwm server logs [name] [-f] [-n 50]
```

| Flag | Description |
|---|---|
| `-f, --follow` | Keep printing new output |
| `-n, --lines <n>` | Number of lines to show, `0` for all (default `50`) |

### server config

```text
nwm server config [-s name]
nwm server config get <key> [-s name]
nwm server config set <key> <value> [-s name]
```

Shows the whole `Config.toml`, prints one value or sets one. Values are TOML; plain words are strings.

### server trust

```text
nwm server trust [name] [--revoke] [--global]
```

Shows, grants or withdraws the approval for `--enable_unsafe_libs`. See [C modules](../servers/c-modules.md).

### server cmd

```text
nwm server cmd [-s name] [--wait 1.5s] <console command>...
```

Sends a command to the console of a server started with `--detach` or `nwm dev` and shows the server's response from the
log, e.g. `nwm server cmd package hotreload my-gamemode`. Everything after the first word belongs to the command, so put
`-s` before it.

### server attach

```text
nwm server attach [name]
```

Follows a background server's output and sends every line you type to its console. `Ctrl+C` detaches, the server keeps
running.

### server remove

```text
nwm server remove <name> [--delete-files]
```

Alias: `rm`. Unregisters the server and removes its containers. `--delete-files` also deletes its folder, volumes and
secrets.

## nwm package and nwm assets

`nwm package` manages packages, `nwm assets` asset packs. Both take `-s/--server`.

| Command | Aliases | Description |
|---|---|---|
| `install <name>... [-e, --enable]` | `add`, `i` | Install from the vault, optionally enable |
| `update [name]...` | `upgrade` | Update the given or all installed resources |
| `list` | `ls` | List installed resources |
| `enable <name>...` | | Load on server start via `Config.toml` |
| `disable <name>...` | | Stop loading on server start |
| `remove <name>...` | `rm`, `uninstall` | Disable and delete |

## nwm nanossharp

```text
nwm nanossharp install [server]
nwm nanossharp update [server]
nwm nanossharp uninstall [server]
```

## nwm recipe

| Command | Description |
|---|---|
| `init [directory \| file] [--name @you/name] [--toml]` | Create a recipe from a template |
| `edit [directory \| file] [--name @you/name] [--toml]` | Open the interactive editor |
| `add [-C path] <statement>...` | Add a statement to a recipe |
| `rm [-C path] <kind> <name>` | Remove an entry (alias `remove`) |
| `show [directory \| file] [--format dsl\|toml]` | Print a recipe |
| `convert [directory \| file] --to dsl\|toml` | Convert between `Recipefile` and `recipe.toml` |
| `validate [directory \| file]` | Check a recipe for mistakes |
| `apply <source> [-s server]` | Apply a recipe to an existing server |
| `upgrade [-s server]` | Apply the server's recipe again with new versions |
| `info <source> [--refresh]` | Show a recipe's content and versions |
| `search [query] [--refresh]` | Search the registry |
| `publish [directory] [--browser]` | Publish to the registry |
| `unpublish <name> [--browser]` | Remove from the registry |

`apply` and `upgrade` take the recipe flags `--var`, `--secret`, `--no-services`, `--no-hooks`, `-y/--yes` and
`--refresh`. `add` and `rm` take `-C/--dir` (default `.`), which has to come before the statement.

## nwm service

All commands take `-s/--server`.

| Command | Description |
|---|---|
| `list` | Services with image, ports and state (alias `ls`) |
| `add <name> --image <image>` | Add a service. Flags: `-p/--port`, `-e/--env`, `-v/--volume` (repeatable), `--start` (default `true`) |
| `up [service]...` | Start services (alias `start`) |
| `down [service]...` | Stop services (alias `stop`) |
| `logs <service> [-f] [-n 50]` | Show a service's output |
| `remove <service>... [--volumes]` | Remove services and their containers (alias `rm`) |

## nwm secret

All commands take `-s/--server`.

| Command | Description |
|---|---|
| `list` | Secret names and lengths (alias `ls`) |
| `get <name>` | Print a value |
| `set <name> [value]` | Set a value, asks when not given |
| `rotate <name> [--length 24]` | Replace with a new random value |
| `remove <name>` | Delete a secret (alias `rm`) |

## nwm dev

```text
nwm dev [directory | file] [-- server arguments] [flags]
```

Develop packages locally with live reload, see [Local development](../local-development.md).

| Flag | Description |
|---|---|
| `-n, --name <name>` | Name of the development server (default `<recipe>-dev`) |
| `-d, --dir <dir>` | Folder of the development server |
| `--local` | Keep the server in `.nwm/` next to the recipe |
| `--reapply` | Apply the whole recipe again even if it did not change |
| `--allow-unsafe-libs` | Approve `--enable_unsafe_libs` without asking |
| `--var <name=value>`, `--secret <name=value>` | Recipe variables and secrets |
| `--no-services`, `--no-hooks` | Skip services or hooks |

## nwm self-update

```text
nwm self-update [--force]
```

Updates nwm to the latest release after verifying its checksum. `--force` reinstalls the current version, and is
required for development builds.

## nwm version

Prints the version, platform and Go version.

## nwm completion

```text
nwm completion bash|zsh|fish|powershell
```

Prints a shell completion script, see [Installation](../getting-started/installation.mdx#shell-completion).
