---
title: Environment variables
---

# Environment variables

## nwm

| Variable | Default | Effect |
|---|---|---|
| `NWM_HOME` | `~/.nwm` | Folder for state, secrets, logs, caches, SteamCMD and default server installs |
| `NWM_REGISTRY_URL` | the official index | URL or file path of the registry `index.json` |
| `NWM_NO_UPDATE_CHECK` | unset | Any value disables the daily update hint |
| `GITHUB_TOKEN` | unset | Used for gist requests to raise GitHub's rate limit |

## Install scripts

| Variable | Default | Effect |
|---|---|---|
| `NWM_VERSION` | `latest` | Release to install, e.g. `0.2.2` |
| `NWM_INSTALL_DIR` | `~/.local/bin` or `%LOCALAPPDATA%\Programs\nwm` | Install folder |

## Passed to processes

| Variable | Where | Value |
|---|---|---|
| Everything in `.env` | the nanos world server process | Values from recipes (`env`) and service presets |
| `NWM_SERVER` | recipe hooks | Name of the server |
| `NWM_SERVER_DIR` | recipe hooks | Server folder |
| `NWM_RECIPE_DIR` | recipe hooks | Folder of the recipe (a cache folder for remote recipes) |
