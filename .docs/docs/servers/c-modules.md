---
title: C modules
---

# C modules and unsafe libraries

A [C module](https://docs.nanos-world.com/docs/core-concepts/packages/c-module) is a package of type `c_module`: a folder
in `Packages/` with a `Package.toml` and the compiled libraries.

```text
Packages/
└── my-module/
    ├── Package.toml      [c_module] modules = ["example"]
    ├── example.dll
    └── libexample.so
```

nwm treats C modules like any other package. `nwm package list` shows their type and `nwm package enable` adds them to
`game.packages`.

## Installing C modules

From the vault or git, use the normal package commands or recipe statements:

```bash
nwm package install my-module --enable
```

```recipefile
package my-module git https://github.com/you/my-module.git ref v1.0.0 enable
```

From prebuilt binaries, use the [`module`](../recipes/recipefile.md#module) statement in a recipe:

```recipefile
module mysql linux https://example.com/libmysql.so windows https://example.com/mysql.dll enable
module tools linux https://example.com/tools-linux.tar.gz windows https://example.com/tools-windows.zip
```

For the current platform nwm downloads the file (or takes it from the recipe folder), extracts archives and installs it
as `Packages/<name>/`. If the files come without a `Package.toml`, nwm writes one and derives the module names from the
libraries: `libexample.so` and `example.dll` both become the module `example`. Archives that contain a single folder are
unpacked into the package folder directly.

## Unsafe libraries

The server only loads C modules when it is started with `--enable_unsafe_libs`. That flag also unlocks unsafe Lua
libraries such as `io` and `os` for every package, which gives them full access to your machine. Otherwise the server
refuses to load the module:

```text
ERROR  Failed loading Package 'mysql'. Reason: 'To be able to load c-module Packages ('mysql'),
       you need to start the server with '--enable_unsafe_libs' (at your own risk).'
```

nwm never adds the flag on its own. You have three ways to start with it:

| Way | When to use |
|---|---|
| `nwm server start dev -- --enable_unsafe_libs` | Once, explicitly |
| `arg --enable_unsafe_libs` in a recipe | The recipe needs it, nwm asks you once |
| `nwm server trust --global` | You always want to allow it |

When C modules are enabled and the flag is missing, `nwm server start`, `nwm package enable` and `nwm recipe apply` print
a warning with these options.

### The approval prompt

If a recipe asks for `--enable_unsafe_libs`, nwm asks before the first start of the server and lists the C modules it
would load:

```text
⚠️  the recipe of rp starts the server with --enable_unsafe_libs
   This lets packages run native code (C modules) and use unsafe Lua libraries
   such as io and os, with full access to this machine. Only allow it for code you trust.
   C modules: mysql
❓ Allow --enable_unsafe_libs for rp? [y] yes, this server · [a] always, all servers · [N] no
```

| Answer | Effect |
|---|---|
| `y` | Allowed for this server and these modules |
| `a` | Allowed for every server from now on |
| anything else | The server starts without the flag, C modules stay disabled |

Without a terminal (scripts, CI) nwm cannot ask, so it starts the server without the flag and prints a warning. Pass
`--allow-unsafe-libs` to `nwm server start` or `restart` to approve it non-interactively.

nwm asks again when new C modules are enabled later, for example after `nwm recipe upgrade`. Approvals are stored in
`~/.nwm/state.toml`, outside the server folder, so a recipe cannot approve itself by writing files.

### Managing approvals

```bash
nwm server trust [name]
nwm server trust [name] --revoke
nwm server trust --global
nwm server trust --global --revoke
```

`trust` shows the approval of a server (when it was given, which modules it covers) or asks for it.
`--revoke` withdraws it. `--global` switches the "always allow" setting.
