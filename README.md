# Pextractor

Portables Windows-Programm, das Titel aus einem Plex-Server ausliest.

- Filme: alle Filmtitel
- Serien: alle Serientitel
- Musik: alle Künstler

Die Liste lässt sich als UTF-8-Textdatei speichern.

## Start

`pextractor.exe` starten. Server-Adresse und Plex-Token eintragen, verbinden, Bibliothek wählen, exportieren.

Den Token findest du in Plex Web über **Infos → XML anzeigen** als Parameter `X-Plex-Token`.

## Bauen

```
go build -trimpath -ldflags "-s -w -H windowsgui" -o pextractor.exe .
```
