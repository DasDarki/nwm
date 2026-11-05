---
title: Creating and updating
---

# Creating and updating servers

## Create a server

```bash
nwm server create <name> [flags]
```

nwm prepares SteamCMD, installs the nanos world dedicated server (Steam app `1936830`) and registers the server.
Names may contain letters, digits, `-` and `_`.

| Flag | Effect |
|---|---|
| `-d, --dir <path>` | Install into this folder instead of `~/.nwm/servers/<name>` |
| `-b, --branch <branch>` | `public` (default) or `bleeding-edge` |
| `--port <port>`, `--query-port <port>` | Set `discover.port` and `discover.query_port` |
| `--display-name <name>` | Set `discover.name`, the name in the server list |
| `--max-players <n>` | Set `general.max_players` |
| `--game-mode <package>`, `--map <package>` | Set `game.game_mode` and `game.map` |
| `--nanossharp` | Install [NanosSharp](./nanossharp.md) afterwards |
| `-r, --recipe <source>` | Set the server up from a [recipe](../recipes/applying.md) |
| `--force` | Install into a folder that is not empty |

When `--recipe` is given, the recipe's Steam branch is used unless you pass `--branch`, and the recipe flags `--var`,
`--secret`, `--no-services`, `--no-hooks`, `--yes` and `--refresh` are available. See
[Applying recipes](../recipes/applying.md).

```bash
nwm server create dev
nwm server create staging --branch bleeding-edge --port 7787 --nanossharp
nwm server create rp --recipe @dasdarki/roleplay --var port=7790
```

## Import an existing installation

Already have a server folder? Register it without reinstalling:

```bash
nwm server import <name> <directory> [--branch public]
```

nwm checks that the folder contains `NanosWorldServer.sh` (Linux) or `NanosWorldServer.exe` (Windows).

## Update a server

```bash
nwm server update [name] [flags]
```

Runs SteamCMD again and installs the newest build of the server's branch. The server has to be stopped.

| Flag | Effect |
|---|---|
| `-b, --branch <branch>` | Switch to `public` or `bleeding-edge` and remember the choice |
| `--no-validate` | Skip SteamCMD's file validation (faster) |

If NanosSharp is installed, nwm reminds you to run `nwm nanossharp update` in case the server version changed.

## List and inspect servers

```bash
nwm server list
nwm server info [name]
```

`list` shows every server with its state (`🟢 running` with process id, or `⚫ stopped`), branch, whether NanosSharp is
installed and its folder. `info` adds the display name, ports, game mode, map, the number of installed packages and
asset packs and the log file.

## Remove a server

```bash
nwm server remove <name> [--delete-files]
```

Unregisters the server and removes its service containers. With `--delete-files` nwm also deletes the server folder,
the service volumes and the server's [secrets](../secrets.md). Without it the files stay where they are.
