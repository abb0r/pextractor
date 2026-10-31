# pExtractor

Portable program that reads titles from a Plex Media Server.

- Movies: every movie title
- TV shows: every series title
- Music: every artist

The list can be saved as a UTF-8 text file.

## Windows

Download `pExtractor.exe` from the [releases](https://github.com/abb0r/pextractor/releases). No installation is required.

## Linux

Download `pExtractor.flatpak` from the same release page, then install it:

```
flatpak remote-add --if-not-exists --user flathub https://flathub.org/repo/flathub.flatpakrepo
flatpak install --user ./pExtractor.flatpak
flatpak run io.github.abb0r.pextractor
```

Flatpak installs the GNOME runtime from Flathub when it is not already present.

## Run

Enter the server address and Plex token, connect, choose a library, then export.

In Plex Web, open a title and choose **Get Info → View XML**. The token is the `X-Plex-Token` parameter in that URL.

## Build

Windows:

```
go build -trimpath -ldflags "-s -w -H windowsgui" -o pExtractor.exe .
```

Linux needs the GNOME Flatpak SDK, Wayland, and EGL development packages because the interface links against them. The release workflow builds `pExtractor.flatpak` with those libraries.
