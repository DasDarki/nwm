# Ziel
Ich möchte ein in Go geschrieben CLI schreiben für das Sandbox Spiel nanos world. Der Name: "nwm" kurz für "nanos world manager". Das Tool soll eine Reihe vereinfachungen mit sich bringen:
1. Einfaches erstellen/updaten von servern. https://docs.nanos-world.com/docs/core-concepts/server-manual/server-installation
2. Erstellen/Verwendung von "Recipes" (vorkonfigurierte, für z.B. lokale Server). Da drinne stehen Server Configurations, Packages, Assets, C-Modules und andere lokale voraussetzungen wie z.B. .envs oder sogar Datenbanken, Redis usw.
3. Manage von anderen Services für Server
4. Manage von Servern
5. Herunterladen/Updaten/Installieren von Packages, Assets, C-Modules usw. (auch von komplexeren Systemen wie NanosSharp) -> siehe: https://docs.nanos-world.com/docs/core-concepts/server-manual/command-line-interface https://docs.nanos-world.com/docs/vault-and-store/store https://docs.nanos-world.com/docs/vault-and-store/vault und https://nanossharp.dasdarki.de/docs/getting-started/installation

## Tech
- Das ganze wird in Go geschrieben mit schönen Animationen, Emojis usw
- Das ganze ist cross plattform für Windows und Linux
- Das ganze soll einen kleinen auto updater haben

## Recipe Hosting
- Können wir dafür wie Nebra Lang (https://github.com/nebra-lang/nebra) eine Package Manager Registry bauen bzw Recipe Registry bauen, die Github als "Hosting" verwendet bzw Index um dann aber einfach nur Git Repos verlinkt
