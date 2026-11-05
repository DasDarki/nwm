---
title: Publishing recipes
---

# Publishing recipes

## Requirements

- A **public GitHub repository** that contains a `Recipefile` or `recipe.toml`, at the root or in a subfolder. One
  repository can hold several recipes in different folders.
- A **scoped name**: `recipe @your-login/name`. The scope is your GitHub login, or an organization you are a **public**
  member of.
- **Version tags** like `v0.1.0`. Without tags users get your default branch.

## Publish

```bash
cd my-recipe
git tag v0.1.0 && git push --tags
nwm recipe publish
```

`nwm recipe publish [directory]` validates the recipe, finds the repository from your `origin` remote (HTTPS or SSH) and
the subfolder from the git root, and warns about uncommitted changes and missing version tags. Then it opens a request in
the registry:

- with the [`gh` CLI](https://cli.github.com) logged in, nwm creates the request issue directly
- otherwise, or with `--browser`, nwm opens a prefilled issue form in your browser

A bot validates the request within a minute or two, adds your recipe to the index, comments on the issue and closes it.
If something is wrong, the bot explains it in a comment and keeps the issue open; edit the issue to try again.

## New versions

Publishing is needed only once. Push a new tag and users get it:

```bash
git tag v0.2.0 && git push --tags
```

`nwm recipe upgrade` on servers that use `@you/name@^0.2` (or no constraint) picks it up.

Run `nwm recipe publish` again only to change the description, keywords, repository or subfolder.

## Unpublish

```bash
nwm recipe unpublish @you/name
```

Opens a request to remove the entry. The repository and its tags stay untouched.

## What the bot checks

| Check | Rule |
|---|---|
| Name | Lowercase `@scope/name`, letters, digits, `-` and `_` |
| Repository | `https://github.com/owner/repo`, public |
| Recipe | `Recipefile` or `recipe.toml` in the given folder, declaring exactly the requested name |
| Scope | Your login, an organization you are a public member of, or you are already an owner of the entry |

The first person to publish into a scope becomes the owner of that entry. Owners can update and unpublish it.
