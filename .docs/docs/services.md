---
title: Services
---

# Services

Most game modes need more than the server: a database, a cache, a search engine. nwm runs these as **Docker containers
that belong to a server**. They are defined in the server's `nwm.toml`, start before the server and are reachable on
`127.0.0.1`.

Services need [Docker](https://docs.docker.com/get-docker/) with a running daemon.

## In a recipe

The quickest way is `use` with a preset:

```recipefile
use postgres
use redis port 16379
```

This adds the containers, generates passwords and writes `DATABASE_URL` and `REDIS_URL` to the server's `.env`, which
the server process receives as environment variables. Lua scripts cannot read those by default, so pass the values on
with [`expose`](./recipes/recipefile.md#expose):

```recipefile
use postgres
expose DATABASE_URL
```

```lua
local url = Server.GetCustomSettings().DATABASE_URL
```

For anything else, describe the container yourself with [`service`](./recipes/recipefile.md#service):

```recipefile
service search getmeili/meilisearch:v1 port 7700 env MEILI_ENV=development volume search-data:/meili_data
env MEILISEARCH_URL http://127.0.0.1:7700
```

## Presets

| Preset | Image | Port | Data volume | Ready check | Writes to `.env` |
|---|---|---|---|---|---|
| `postgres` | `postgres:17` | 5432 | `/var/lib/postgresql/data` | `pg_isready` | `DATABASE_URL` |
| `mysql` | `mysql:8.4` | 3306 | `/var/lib/mysql` | `mysqladmin ping` | `DATABASE_URL` |
| `mariadb` | `mariadb:11` | 3306 | `/var/lib/mysql` | `mariadb-admin ping` | `DATABASE_URL` |
| `redis` | `redis:7-alpine` | 6379 | `/data` | `redis-cli ping` | `REDIS_URL` |
| `valkey` | `valkey/valkey:8-alpine` | 6379 | `/data` | `valkey-cli ping` | `REDIS_URL` |
| `mongo` | `mongo:8` | 27017 | `/data/db` | `mongosh` ping | `MONGODB_URL` |
| `rabbitmq` | `rabbitmq:4-management` | 5672 | `/var/lib/rabbitmq` | `rabbitmq-diagnostics` | `RABBITMQ_URL` |
| `meilisearch` | `getmeili/meilisearch:v1` | 7700 | `/meili_data` | port | `MEILISEARCH_URL` |
| `minio` | `minio/minio` | 9000 | `/data` | port | `S3_ENDPOINT` |

The values look like this (the port follows `port`, the password is the service's secret):

```text
DATABASE_URL=postgres://nanos:<password>@127.0.0.1:5432/nanos
DATABASE_URL=mysql://root:<password>@127.0.0.1:3306/nanos
REDIS_URL=redis://127.0.0.1:6379
MONGODB_URL=mongodb://nanos:<password>@127.0.0.1:27017
RABBITMQ_URL=amqp://guest:guest@127.0.0.1:5672
MEILISEARCH_URL=http://127.0.0.1:7700
S3_ENDPOINT=http://127.0.0.1:9000
```

Databases use the user `nanos` (MySQL and MariaDB: `root`) and the database `nanos`. Their password is a
[secret](./secrets.md) named after the service, e.g. `postgres_password`, generated on the first apply.

- `use postgres:16` keeps your tag but still applies the preset
- `use postgres as main-db` names the service, volume and secret `main-db`, `main-db-data`, `main_db_password`
- `use postgres port 15432` publishes on host port 15432 and adjusts `DATABASE_URL`
- `use postgres env POSTGRES_DB=game` adds or overrides container variables
- an `env` entry that already exists in the recipe is not overwritten
- unknown images become a plain service without defaults

## Ports, volumes and readiness

**Ports** bind to `127.0.0.1` so your databases are not reachable from the network. To expose one on purpose, include
the address: `port 0.0.0.0:5432:5432`. `port 15432:5432` publishes the container port 5432 on host port 15432.

**Volumes** written as `name:/path` are Docker volumes managed by nwm and named `nwm-<server>-<name>`, so data survives
container updates. Volumes written as `./dir:/path` mount a folder relative to the server folder.

**Readiness**: before the server starts, nwm waits up to two minutes until each service is ready. With a `ready` check
it runs the command inside the container until it succeeds; otherwise it waits until the first published port accepts
connections. `nowait` skips the wait. The explicit checks matter for databases: Docker accepts connections on the port
before the database inside has finished its first start.

## Lifecycle

Containers are named `nwm-<server>-<service>`, labelled with the server name and restart automatically unless stopped.
nwm remembers a hash of each definition and recreates the container only when the definition changes; otherwise it just
starts the existing one.

| When | What happens |
|---|---|
| `nwm server start` | Services are started first (skip with `--no-services`) |
| `nwm server stop --services` | Services are stopped after the server |
| `nwm recipe apply` / `upgrade` | Services are created or recreated and started (skip with `--no-services`) |
| `nwm server remove` | Containers are removed, with `--delete-files` also their volumes |

## Commands

```bash
nwm service list [-s server]
nwm service add <name> --image <image> [-p port] [-e KEY=value] [-v volume] [--start=false] [-s server]
nwm service up [service]... [-s server]
nwm service down [service]... [-s server]
nwm service logs <service> [-f] [-n 50] [-s server]
nwm service remove <service>... [--volumes] [-s server]
```

Alias: `nwm svc`.

`add` defines a service outside of recipes and starts it right away. `up` and `down` start and stop all services or the
given ones. `remove` deletes the containers and the definitions; `--volumes` also deletes their data.

```bash
nwm svc add cache --image redis:7-alpine --port 6379
nwm svc add db --image postgres:16 --port 5432 --env POSTGRES_PASSWORD=dev --volume data:/var/lib/postgresql/data
nwm svc logs db -f
```
