# 🌍 nwm - nanos world manager

A cross-platform CLI (Windows and Linux) that takes the busywork out of running [nanos world](https://nanos-world.com) servers.

## Install

**Linux**

```sh
curl -fsSL https://raw.githubusercontent.com/DasDarki/nwm/main/install.sh | sh
```

**Windows** (PowerShell)

```powershell
irm https://raw.githubusercontent.com/DasDarki/nwm/main/install.ps1 | iex
```

The installers verify the download against `checksums.txt`, need no admin rights and add nwm to your `PATH`
(Linux: `~/.local/bin`, registered in `.profile`, `.bashrc`, `.zshrc` and fish; Windows: `%LOCALAPPDATA%\Programs\nwm`, registered in the user `PATH`).
Set `NWM_VERSION` to install a specific release and `NWM_INSTALL_DIR` to change the location.

Alternatively download the binary from the [latest release](https://github.com/DasDarki/nwm/releases/latest).
Afterwards nwm keeps itself up to date:

```sh
nwm self-update
```

## Quick start

```sh
nwm server create dev --port 7777 --nanossharp
nwm package install sandbox --enable
nwm server start dev
```

## Commands

| Command | Description |
|---|---|
| `nwm server create <name>` | Install a server via SteamCMD (downloaded automatically) |
| `nwm server import <name> <dir>` | Register an existing installation |
| `nwm server update [name]` | Update to the latest build, `--branch bleeding-edge` switches branches |
| `nwm server list` / `info` | Show servers and their status |
| `nwm server start [name]` | Run in the foreground, or in the background with `-d` |
| `nwm server stop` / `restart` / `logs -f` | Control background servers |
| `nwm server config [get\|set]` | Read and edit `Config.toml` |
| `nwm package install\|update\|list\|remove\|enable\|disable` | Manage packages from the vault |
| `nwm assets install\|update\|list\|remove\|enable\|disable` | Manage asset packs from the vault |
| `nwm nanossharp install\|update\|uninstall` | Manage [NanosSharp](https://nanossharp.dasdarki.de) |
| `nwm recipe init\|validate\|apply\|upgrade\|info\|search` | Work with recipes |
| `nwm recipe publish\|unpublish` | Share recipes through the registry |
| `nwm service list\|add\|up\|down\|logs\|remove` | Manage Docker services of a server |
| `nwm self-update` | Update nwm itself |

Commands that target a server pick it by name, by the current directory, or automatically when only one server exists.

## Recipes

A recipe is a `recipe.toml` (plus any files it references) that describes a complete server setup.
Create one with `nwm recipe init`, then use it:

```sh
nwm server create dev --recipe ./my-recipe --var port=7790
nwm recipe apply @dasdarki/roleplay@^1 -s dev
nwm recipe upgrade -s dev
```

### Sources

| Source | Example |
|---|---|
| Registry | `@dasdarki/roleplay`, `@dasdarki/roleplay@^1`, `roleplay@1.2.0` |
| GitHub | `github:owner/repo`, `github:owner/repo/sub/dir@v1.2.0` |
| Git URL | `https://host/repo.git#v1`, `git+https://host/repo.git//sub/dir#^1` |
| Local | `./my-recipe`, `file:/path/to/recipe` |

Versions are git tags. Constraints: exact (`1.2.3`), `^1`, `^1.2`, `~1.2`, `>=1.0`, a branch or tag name, or nothing for the newest release.

### Format

```toml
[recipe]
name = "@you/roleplay"
version = "1.0.0"
description = "Roleplay development server"
keywords = ["roleplay", "postgres"]

[server]
branch = "public"
nanossharp = true
args = ["--log_level", "2"]

[variables]
port = 7777
db_password = "dev"

[config.discover]
name = "RP ${server.name}"
port = "${port}"

[config.custom_settings]
spawn_cars = true

[[package]]
name = "sandbox"
enable = true

[[package]]
name = "roleplay-core"
git = "https://github.com/you/roleplay-core.git"
ref = "v2.1.0"
enable = true

[[package]]
name = "dev-tools"
path = "packages/dev-tools"

[[assets]]
name = "nanos-world"

[[module]]
name = "mysql"
linux = "https://example.com/mysql.so"
windows = "https://example.com/mysql.dll"

[[file]]
source = "files/NanosSharp.toml"
target = "NanosSharp.toml"
template = true

[[service]]
name = "db"
image = "postgres:16"
ports = ["5432"]
env = { POSTGRES_PASSWORD = "${db_password}" }
volumes = ["data:/var/lib/postgresql/data"]

[env]
DATABASE_URL = "postgres://postgres:${db_password}@127.0.0.1:5432/postgres"

[[hook]]
run = ["dotnet", "build"]
dir = "Packages/roleplay-core"
```

| Section | Effect |
|---|---|
| `[server]` | Steam branch, NanosSharp installation and default launch arguments |
| `[variables]` | Defaults for `${name}` placeholders, overridable with `--var name=value`. Built-ins: `server.name`, `server.path`, `os` |
| `[config]` | Merged into `Config.toml`. A value that is only a placeholder keeps its type |
| `[[package]]`, `[[assets]]` | From the vault (name only), from git (`git`, `ref`, `subdir`) or from the recipe (`path`). `enable` adds them to `Config.toml` |
| `[[module]]` | C modules copied into `Modules/`, per platform, from a URL or recipe path. Archives are extracted |
| `[[file]]` | Files or folders copied into the server. `template = true` expands placeholders |
| `[[service]]` | Docker containers started with the server. Ports bind to `127.0.0.1` unless an IP is given. Named volumes are managed by nwm, paths are relative to the server |
| `[env]` | Written to the server's `.env` and passed to the server process |
| `[[hook]]` | Commands run after applying. Hooks from remote recipes ask for confirmation unless `--yes` is given |

The applied recipe, its variables, launch arguments and services are stored in the server's `nwm.toml`.

## Registry

Published recipes are indexed in [DasDarki/nwm-registry](https://github.com/DasDarki/nwm-registry). The registry only
stores pointers to public GitHub repositories, versions stay git tags. `nwm recipe publish` opens a request
issue (through the `gh` CLI when available) that a workflow validates and merges. Set `NWM_REGISTRY_URL` to use
another index (URL or local file).

## Requirements

- Recipes from git and git packages need `git`
- Services need Docker
- SteamCMD is installed automatically

## Files

nwm keeps its state in `~/.nwm` (override with `NWM_HOME`): SteamCMD, the server registry, default server installs, background logs and caches. Set `NWM_NO_UPDATE_CHECK=1` to disable the daily update hint.

## Releasing

Push a `vX.Y.Z` tag. GoReleaser builds `nwm_<os>_<arch>` binaries plus `checksums.txt`, which the self-updater verifies before installing.
