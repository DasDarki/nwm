---
title: Configuration
---

# Server configuration

Every nanos world server reads `Config.toml` from its folder. nwm can show and change it without opening an editor.

```bash
nwm server config [-s name]
nwm server config get <key> [-s name]
nwm server config set <key> <value> [-s name]
```

Keys use dots for sections, for example `discover.port` or `general.max_players`.

```bash
nwm server config get discover.port -s dev
nwm server config set discover.port 7787 -s dev
nwm server config set discover.name '"My Server"' -s dev
nwm server config set game.packages '["sandbox", "my-script"]' -s dev
nwm server config set custom_settings.spawn_cars true -s dev
```

## How values are read

The value is parsed as a TOML value: `7787` becomes a number, `true` a boolean, `["a", "b"]` a list and `"text"` a string.
Anything that is not valid TOML, such as a plain word, is stored as a string.

## Complete files

nanos world refuses to start when whole sections are missing from `Config.toml`
(`Key "general" not found in the table`). Whenever nwm writes the file it fills in every missing key with the server's
default value and keeps your own values. This also applies to recipes and `--port`, `--display-name` and friends on
`nwm server create`.

The server itself rewrites `Config.toml` on startup and drops comments, so keep notes somewhere else, for example in a
[recipe](../recipes/overview.md).

## Default configuration

These are the keys nwm fills in, with their defaults:

```toml
[discover]
name = "nanos world server"
description = ""
language = "global"
ip = "0.0.0.0"
port = 7777
query_port = 7778
announce = true
dedicated_server = true

[general]
max_players = 64
password = ""
token = ""
banned_ids = []

[game]
map = "default-blank-map"
game_mode = ""
packages = []
assets = []
loading_screen = ""

[custom_settings]

[debug]
log_level = 1
async_log = true
profiling = false

[optimization]
max_tick_rate = 30
max_sync_rate = 30
max_send_rate = 2048
max_file_transfer_rate = 2048
distance_optimization = 4
```

See the [nanos world server configuration docs](https://docs.nanos-world.com/docs/core-concepts/server-manual/server-configuration)
for what each key does.
