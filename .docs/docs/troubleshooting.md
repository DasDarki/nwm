---
title: Troubleshooting
---

# Troubleshooting

When an external program such as SteamCMD, git or Docker fails, nwm prints its last lines of output below the error.
Start there.

## SteamCMD

**SteamCMD does not start on Linux.** It is a 32-bit program. Install the 32-bit C runtime, for example on Debian and
Ubuntu: `sudo dpkg --add-architecture i386 && sudo apt update && sudo apt install lib32gcc-s1`.

**"SteamCMD finished without confirming the installation".** SteamCMD updates itself on its first run and sometimes
exits early; nwm already retries once. Run the command again, or `nwm server update` for an existing server.

## Starting servers

**"server stopped right after starting".** The server crashed within two seconds. Read `nwm server logs <name>` or start
it in the foreground with `nwm server start <name>` to see the output.

**`Key "general" not found in the table at Config.toml`.** The `Config.toml` is missing whole sections. Any
`nwm server config set` fills them in, for example `nwm server config set discover.port 7777`.

**"server is already running".** nwm found a process for this server. Stop it with `nwm server stop`. Entries of
processes that no longer exist are cleaned up automatically.

**C modules do not load.** The server needs `--enable_unsafe_libs`, see [C modules](./servers/c-modules.md#unsafe-libraries).

**NanosSharp disabled itself after an update.** Run `nwm nanossharp update` to get a release that matches the server.

## Services

**"docker daemon is not reachable".** Start Docker, and on Linux make sure your user may use it
(`sudo usermod -aG docker $USER`, then log in again).

**"port is already allocated".** Another program or container uses the port. Pick another host port, e.g.
`use postgres port 15432`, and apply the recipe again.

**"service was not ready within 2m0s".** Look at the container with `nwm service logs <service>`. A common cause is a
password change after the data volume was created, see [rotating passwords](./secrets.md#commands).

## Recipes

**"unknown variables: ...".** A `${...}` placeholder has no value. Define it with `var`, pass `--var name=value`, or
declare the secret with `secret name`.

**"secret ... needs a value, pass --secret ...".** A `prompt` secret was needed but nwm had no terminal to ask. Pass it
with `--secret name=value`.

**"single-file recipes cannot reference local files".** Recipes loaded from a URL or gist have no folder around them.
Publish the recipe and its files as a git repository instead, see [Sources](./recipes/sources.md#file-urls-and-gists).

**"... is not a known Config.toml key".** Usually a typo, for example `discover.max_players` instead of
`general.max_players`. Custom values belong under `custom_settings.`.

**Gist requests fail with 403.** GitHub's rate limit for anonymous API requests was reached. Set `GITHUB_TOKEN`.

**"git is required".** Install git; it is needed for git recipes and git packages.

## Completion

**Package names do not complete.** The store catalog could not be downloaded. nwm uses a cached copy when there is one;
check your connection and try again.

## Getting help

Open an issue on [GitHub](https://github.com/DasDarki/nwm/issues) with the command you ran, its output and
`nwm version`.
