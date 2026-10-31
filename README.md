# Pextractor

Portable Windows program that reads titles from a Plex Media Server.

- Movies: every movie title
- TV shows: every series title
- Music: every artist

The list can be saved as a UTF-8 text file.

## Run

Start `pextractor.exe`. Enter the server address and Plex token, connect, choose a library, then export.

In Plex Web, open a title and choose **Get Info → View XML**. The token is the `X-Plex-Token` parameter in that URL.

## Build

```
go build -trimpath -ldflags "-s -w -H windowsgui" -o pextractor.exe .
```
