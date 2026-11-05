---
title: Editing recipes
---

# Editing recipes

You can write a `Recipefile` in any text editor. nwm adds three ways that understand the format: a starter template,
an interactive editor with completion and single statements from your shell.

## Start a recipe

```bash
nwm recipe init [directory | file] [--name @you/name] [--toml]
```

Creates a `Recipefile` with a small working example. Pass a file name to create a named recipe next to others, or
`--toml` to create `recipe.toml`.

```bash
nwm recipe init --name @you/roleplay
nwm recipe init staging.Recipefile
```

## The interactive editor

```bash
nwm recipe edit [directory | file] [--name @you/name] [--toml]
```

The editor shows the current recipe at the top and a command line at the bottom. Every [statement](./recipefile.md) is a
command: type `package sandbox enable` and press Enter to add it. If no recipe exists yet, a new one is created and
written on `save`.

### Completion

Press `Tab` to complete the word under the cursor, again to cycle through the suggestions. nwm completes:

- commands and their options
- package and asset pack names from the nanos world store, with title, type and description
- [service presets](../services.md#presets) for `use` and images for `service`
- `Config.toml` keys with their default or current value
- files and folders inside the recipe
- existing entries for `remove`

The store catalog is downloaded on first use and cached for a day.

### Keys

| Key | Action |
|---|---|
| `Enter` | Run the command |
| `Tab` / `Shift+Tab` | Next / previous completion |
| `↑` / `↓` | Command history |
| `PgUp` / `PgDn` | Scroll the preview |
| `Esc` | Clear the line |
| `Ctrl+C` | Quit, press twice if there are unsaved changes |

### Editor commands

| Command | Action |
|---|---|
| `remove <kind> <name>` (`rm`) | Remove an entry, e.g. `remove package sandbox`, `remove config discover.port` |
| `undo` (`u`) | Revert the last change |
| `validate` | Check the recipe, including referenced files |
| `save [toml\|dsl]` (`w`) | Write the file, optionally converting it |
| `wq` | Save and quit |
| `help [command]` (`?`) | Show all statements or one statement |
| `quit` (`q`) | Quit, refuses when there are unsaved changes |
| `quit!` (`q!`) | Quit and discard changes |

## From the shell

Every statement also works as a single command, with shell completion for store packages and the rest:

```bash
nwm recipe add package sandbox enable
nwm recipe add use postgres
nwm recipe add config discover.port 7790
nwm recipe add secret steam_token prompt
nwm recipe rm package sandbox
nwm recipe rm service redis
```

`-C/--dir` selects another folder or a named recipe file. It has to come before the statement, because everything after
the first word belongs to the statement:

```bash
nwm recipe add -C staging.Recipefile arg --enable_unsafe_libs
```

## Show, convert and validate

```bash
nwm recipe show [directory | file] [--format dsl|toml]
nwm recipe convert [directory | file] --to dsl|toml
nwm recipe validate [directory | file]
```

`show` prints the recipe, optionally in the other format. `convert` rewrites it in the other format and removes the old
file (`staging.Recipefile` becomes `staging.toml` and back). `validate` checks names, options, duplicate entries and that
every referenced file exists in the recipe.
