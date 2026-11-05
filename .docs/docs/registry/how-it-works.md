---
title: How the registry works
---

# How the registry works

The registry is a regular GitHub repository, [DasDarki/nwm-registry](https://github.com/DasDarki/nwm-registry). It stores
pointers, not recipes:

```text
nwm-registry/
├── index.json                    all entries, read by nwm
├── recipes/
│   └── <scope>/<name>.json       one file per recipe, written by the bot
├── tools/registry/               the validator (Go)
└── .github/
    ├── ISSUE_TEMPLATE/           publish and unpublish forms
    └── workflows/process.yml     runs the validator on new issues
```

## Request flow

1. `nwm recipe publish` opens an issue titled `[publish] @scope/name` (or `[unpublish] ...`) with the fields of the
   issue form.
2. The workflow runs `tools/registry` with the issue body and author.
3. The validator checks the name and scope, reads the recipe from the repository through the GitHub API and writes
   `recipes/<scope>/<name>.json`.
4. It rebuilds `index.json` from all entry files.
5. The workflow commits the change, comments with the result and closes the issue.

## Index format

```json
{
  "version": 1,
  "recipes": [
    {
      "name": "@dasdarki/roleplay",
      "description": "Roleplay development server",
      "repository": "https://github.com/DasDarki/roleplay-recipe",
      "subdir": "recipes/roleplay",
      "keywords": ["roleplay", "postgres"],
      "owners": ["dasdarki"],
      "published_at": "2026-11-05T21:10:49Z",
      "updated_at": "2026-11-05T21:10:49Z"
    }
  ]
}
```

| Field | Meaning |
|---|---|
| `name` | Scoped recipe name |
| `repository` | Public git repository with the recipe |
| `subdir` | Folder of the recipe inside the repository, empty for the root |
| `description`, `keywords` | Shown and searched by `nwm recipe search` |
| `owners` | GitHub logins allowed to update or unpublish the entry |

nwm only needs `index.json`. Any web server or file that serves this format works as a private registry through
`NWM_REGISTRY_URL`.

## Versions

The registry never stores versions. nwm lists the tags of the repository with `git ls-remote` and picks the best match
for the requested constraint, see [Sources and versions](../recipes/sources.md#versions).
