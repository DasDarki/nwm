# nwm documentation

The website for [nwm](https://github.com/DasDarki/nwm), built with [Docusaurus](https://docusaurus.io) and hosted at
[nwm.dasdarki.de](https://nwm.dasdarki.de).

```bash
npm ci
npm start
npm run build
```

`npm start` and `npm run build` copy `install.sh` and `install.ps1` from the repository root into `static/`, so the
installers are served as `https://nwm.dasdarki.de/install.sh` and `/install.ps1`.

## Cloudflare Pages

| Setting | Value |
|---|---|
| Framework preset | Docusaurus |
| Root directory | `.docs` |
| Build command | `npm ci && npm run build` |
| Build output directory | `build` |
| Environment variable | `NODE_VERSION` = `22` |
