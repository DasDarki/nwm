---
title: Running servers
---

# Running servers

## Start

```bash
nwm server start [name] [-- server arguments] [flags]
```

| Flag | Effect |
|---|---|
| `-d, --detach` | Run in the background and write the output to `~/.nwm/logs/<name>.log` |
| `--no-services` | Do not start the server's [Docker services](../services.md) |
| `--allow-unsafe-libs` | Approve `--enable_unsafe_libs` from the recipe without asking, see [C modules](./c-modules.md) |

Without `--detach` the server runs in your terminal, including its interactive console. Press `Ctrl+C` to stop it;
nwm passes the server's exit code on.

Before the server starts nwm:

1. starts the server's services and waits until they are ready,
2. adds the launch arguments stored in the server's `nwm.toml` (usually from a recipe),
3. passes the variables from the server's `.env` file to the process.

Everything after `--` is passed to the server unchanged:

```bash
nwm server start dev -- --log_level 2 --max_tick_rate 60
```

On Linux nwm starts `NanosSharp.sh` when NanosSharp is installed and `NanosWorldServer.sh` otherwise. On Windows it
starts `NanosWorldServer.exe`.

### Background servers

```bash
nwm server start dev -d
nwm server logs dev -f
```

nwm waits two seconds after starting and reports an error (with the path of the log) if the server stopped right away.
The log file is replaced on each start.

## Stop and restart

```bash
nwm server stop [name] [--timeout 15s] [--services]
nwm server restart [name] [-- server arguments] [--timeout 15s]
```

`stop` asks the server to shut down and kills it after the timeout. On Linux the whole process group is signalled.
On Windows the server is terminated directly, because a background process has no console to receive `Ctrl+C`.
With `--services` the server's Docker services are stopped as well.

`restart` stops the server if it is running and starts it again in the background.

## Logs

```bash
nwm server logs [name] [-f] [-n 50]
```

Shows the last lines of a background server's output. `-f` keeps printing new lines, `-n 0` prints the whole log.
Servers started in the foreground print directly to your terminal and write no log file.

## Status

`nwm server list` and `nwm server info` show whether a server is running. nwm tracks running servers in
`~/.nwm/run/<name>.json` and cleans up stale entries automatically when a process is gone.
