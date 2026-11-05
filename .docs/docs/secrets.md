---
title: Secrets
---

# Secrets

Passwords and tokens do not belong into recipes, which you may share or publish, nor into server folders, which you may
copy or commit. nwm keeps them in a **secret store per server**:

```text
~/.nwm/secrets/<server>.toml
```

The folder is readable only by you (mode `0700`, files `0600` on Linux; on Windows it is protected by your user
profile's permissions).

## Declaring secrets in a recipe

```recipefile
secret api_key
secret session_key random 64
secret steam_token prompt

env API_KEY ${secret.api_key}
env STEAM_TOKEN ${secret.steam_token}
```

| Form | When the value is created |
|---|---|
| `secret name` | Random, 24 characters, on the first apply |
| `secret name random <length>` | Random with 8 to 256 characters |
| `secret name prompt` | nwm asks for it on the first apply, without echoing your input |

Random values use letters and digits without look-alikes (`0/O`, `1/l/I`), so they are safe in URLs and config files.

Once a secret exists nwm keeps it: re-applying or upgrading a recipe never changes a password your database already
uses. Pass `--secret name=value` to `nwm server create --recipe`, `nwm recipe apply` or `nwm recipe upgrade` to set a
value explicitly, for example in scripts where nwm cannot ask.

[Service presets](./services.md#presets) declare their password secrets on their own: `use postgres` creates
`postgres_password`.

## Using secrets

`${secret.name}` works everywhere placeholders do. Two places are special:

- in **service environment variables** the placeholder stays in `nwm.toml` and is resolved only when the container
  starts, so the server folder never contains the password in its configuration
- in **`.env`** the real value is written, because the server process needs it

## Commands

```bash
nwm secret list [-s server]
nwm secret get <name> [-s server]
nwm secret set <name> [value] [-s server]
nwm secret rotate <name> [--length 24] [-s server]
nwm secret remove <name> [-s server]
```

Aliases: `nwm sec`, `nwm secrets`.

| Command | Effect |
|---|---|
| `list` | Shows names and lengths, never the values |
| `get` | Prints a value, e.g. for `psql` or a database GUI |
| `set` | Sets a value, asks for it (hidden) when not given |
| `rotate` | Replaces a value with a new random one |
| `remove` | Deletes a secret |

After `set` or `rotate`, restart the services or apply the recipe again so containers and `.env` get the new value.

:::caution Rotating database passwords
Databases store their password when their data volume is created. Rotating the secret afterwards changes what nwm
passes to the container, but not the password inside the existing database. Change it in the database as well, or
remove the volume to start fresh.
:::

`nwm server remove <name> --delete-files` deletes the server's secrets together with its files.
