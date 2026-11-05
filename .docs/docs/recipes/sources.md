---
title: Sources and versions
---

# Sources and versions

Everywhere nwm takes a recipe (`--recipe`, `recipe apply`, `recipe info`) you can point it at one of these sources:

| Source | Examples |
|---|---|
| Registry | `@dasdarki/roleplay`, `@dasdarki/roleplay@^1`, `roleplay@1.2.0` |
| GitHub | `github:owner/repo`, `github:owner/repo/sub/dir@v1.2.0` |
| Git URL | `https://host/repo.git#v1`, `git+https://host/repo.git//sub/dir#^1`, `git@github.com:owner/repo.git#main` |
| Local folder | `.`, `./my-recipe`, `/path/to/recipe`, `file:/path/to/recipe` |
| Local file | `./Recipefile`, `./staging.Recipefile`, `./prod.recipe`, `./dev.toml` |
| File URL | `https://example.com/rp.Recipefile`, `https://example.com/recipe.toml` |
| Gist | `gist:you/<id>`, `gist:<id>/staging.Recipefile`, `gist:<id>@<revision>` |

## Registry

`@scope/name` looks the recipe up in the [registry](../registry/using.md), which points to a git repository. Unscoped
names like `roleplay` work when only one scope has a recipe of that name.

## Git repositories

nwm needs `git` for these. In the URL forms, `#` starts the version and `//` separates a subfolder: in
`git+https://host/repo.git//recipes/rp#^1`, the recipe lives in `recipes/rp` of the repository. The `git+` prefix forces
an `https://` URL to be treated as git.

Checkouts of tags are cached in `~/.nwm/cache/recipes` and reused, branches are fetched again.

## Local folders and files

A folder must contain exactly one of `Recipefile` and `recipe.toml`. A file can have any name; names ending in `.toml`
are read as TOML, all others as Recipefile. Paths in the recipe (`file`, `package ... path`, local modules) are relative
to the folder of the file.

## File URLs and gists

A URL is treated as a single file when it ends with `Recipefile`, `.Recipefile`, `.recipe` or `.toml`. Other URLs are
git repositories.

For a gist nwm uses the GitHub API. If the gist contains one recipe file it is picked automatically, otherwise name it:
`gist:<id>/staging.Recipefile`. Append `@<revision>` to pin a revision. Without authentication GitHub allows 60 API
requests per hour; set `GITHUB_TOKEN` to raise that.

Single files have no folder around them, so they cannot use `file`, `package ... path` or modules from local paths.
nwm refuses such recipes and lists the offending paths. Everything else works, including git packages and modules from
URLs.

## Versions

For registry and git sources the version is a git tag. Write a constraint after `@` (registry, `github:`) or `#` (git
URLs):

| Constraint | Picks |
|---|---|
| nothing, `latest`, `*` | The newest release tag; the default branch if there are no version tags |
| `1.2.3`, `v1.2.3` | Exactly this version |
| `1.2` | The newest `1.2.x` |
| `1` | The newest `1.x` |
| `^1.2` | The newest `>= 1.2.0` and `< 2.0.0` |
| `^0.3` | The newest `0.3.x` |
| `~1.2` | The newest `1.2.x` |
| `>=1.0` | The newest `>= 1.0.0` |
| `main`, `dev`, `v2-beta` | This branch or tag as it is |

Pre-releases such as `v2.0.0-rc.1` are only picked when you ask for them exactly.

File URLs and gists have no tags. Their version is the content hash (`sha256:…`) or the gist revision (`rev:…`), which is
how `nwm recipe upgrade` notices changes.

## Inspect a recipe

```bash
nwm recipe info <source>
```

Shows the description, authors, branch, what the recipe contains, its variables and, for git and registry sources, the
ten newest versions.
