---
title: Using the registry
---

# Using the registry

The nwm registry is a list of published recipes, hosted on GitHub at
[DasDarki/nwm-registry](https://github.com/DasDarki/nwm-registry). There is no package server: every entry points to a
public git repository, and every version is a git tag in that repository.

## Search

```bash
nwm recipe search [query] [--refresh]
```

Searches names, descriptions and keywords. Without a query it lists every recipe.

```bash
nwm recipe search roleplay
nwm recipe search postgres mysql
```

## Install

Use the name wherever nwm takes a recipe:

```bash
nwm server create rp --recipe @dasdarki/roleplay
nwm server create rp --recipe @dasdarki/roleplay@^1
nwm recipe apply roleplay -s rp
nwm recipe info @dasdarki/roleplay
```

Names are scoped like `@owner/name`. Leaving out the scope works when only one owner published a recipe with that name.
Versions use the [constraint syntax](../recipes/sources.md#versions). nwm checks that the repository really contains a
recipe with the published name before applying it.

Recipes from the registry are remote: their [hooks](../recipes/recipefile.md#hook) need your confirmation and
`--enable_unsafe_libs` needs your [approval](../servers/c-modules.md#the-approval-prompt).

## Caching

nwm downloads the registry index on first use and caches it for a day in `~/.nwm/cache/registry/index.json`. A name that
is not in the cached index triggers a refresh, `--refresh` forces one. When GitHub is unreachable nwm falls back to the
cached copy.

## Using another registry

Set `NWM_REGISTRY_URL` to the URL of another `index.json`, or to a local file, for example to test changes or run a
private registry for your team:

```bash
NWM_REGISTRY_URL=https://example.com/nwm/index.json nwm recipe search
```

The format is described in [How the registry works](./how-it-works.md#index-format).
