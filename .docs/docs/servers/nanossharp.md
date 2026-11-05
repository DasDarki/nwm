---
title: NanosSharp
---

# NanosSharp

[NanosSharp](https://nanossharp.dasdarki.de) adds C# scripting to nanos world servers. nwm wraps its official installer.

```bash
nwm nanossharp install [server]
nwm nanossharp update [server]
nwm nanossharp uninstall [server]
```

Aliases: `nwm n`, `nwm ns`, `nwm sharp`, `nwm cs`. The server has to be stopped.

nwm downloads the latest `nanossharp-install` for your platform (Linux x64 or Windows x64) and runs it inside the server
folder with `server --yes`, so it never waits for input. The installer verifies its download, installs the .NET runtime
if needed and keeps your `NanosSharp.toml` on uninstall.

You can also install NanosSharp while creating a server, or from a recipe:

```bash
nwm server create dev --nanossharp
```

```recipefile
nanossharp
```

## Starting servers with NanosSharp

On Linux NanosSharp ships its own launcher, `NanosSharp.sh`. nwm uses it automatically for `nwm server start` once
NanosSharp is installed. On Windows NanosSharp loads through `version.dll`, so the server starts as usual.

`nwm server list` shows a `✓` in the `C#` column for servers with NanosSharp.

:::tip After server updates
NanosSharp checks the server version on start and disables itself when they do not match. After
`nwm server update`, run `nwm nanossharp update` to get a matching release.
:::
