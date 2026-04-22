# MusicBot

***English** · [Русский](README_ru.md)*

Multi-instance Discord music bot written in Go. Supports Spotify, Yandex Music, SoundCloud, YouTube, and DAVE.

---

## Features

- Playback from **Spotify**, **Yandex Music**, **SoundCloud**, **YouTube**, and any source supported by Lavalink.
- Track and album search via `/search` with one-click play from the results.
- Queue with `off / track / queue` repeat modes, pagination, and per-item removal.
- Slash commands + an interactive button-based player (pause, next, volume, repeat, stop).
- Auto-leave from an empty voice channel after 5 minutes.
- Queue recovery after restart — state is persisted in PostgreSQL.
- Support [godave](https://github.com/disgoorg/godave).

## Architecture

The bot runs as two roles:

- **DJ** — the single bot that receives users' slash commands. Owns the shared node registry and the database.
- **Node** — a pool of separate Discord bots that actually occupy voice channels. The DJ picks a free Node per `(guild, channel)` pair and delegates playback to it through Lavalink.

This split allows simultaneous playback in multiple voice channels of the same guild — concurrency is bounded by the number of configured Node bots.

## Requirements

| | |
|---|---|
| **Go** | 1.26+ |
| **PostgreSQL** | 13+ |
| **Lavalink** | 4.x — https://lavalink.dev/ |
| **libdave** | built from source, see below |
| **Build** | `CGO_ENABLED=1` + C/C++ toolchain (gcc/clang, make, cmake) |

## Setup

### 1. Clone

```bash
git clone https://gitlab.com/yokkkoso/musicbot musicbot
cd musicbot
```

### 2. libdave (required before building)

The project imports `github.com/disgoorg/godave/golibdave`, which dynamically links against the native **libdave** library — the reference implementation of Discord E2EE. There is no `go install` equivalent: you need to run the install script from the [godave](https://github.com/disgoorg/godave) repo, which fetches libdave sources and builds `.so` / `.dylib` / `.dll`:

```bash
# Linux / macOS / WSL
./libdave_install.sh v1.1.0

# Windows
.\libdave_install.ps1 v1.1.0
```

### 3. Lavalink

Run your own Lavalink server following the [official documentation](https://lavalink.dev/), along with the [lavasrc](https://github.com/topi314/LavaSrc) and [lavasearch](https://github.com/topi314/LavaSearch) plugins.

At minimum: Lavalink must be reachable at `host:port` with a password matching a `[[lavalink_nodes]]` entry.

### 4. Configuration

```bash
cp configs/config.example.toml configs/config.toml
```

Fill in the fields:

| Field | Purpose |
|---|---|
| `dj_token` | DJ bot token |
| `color` | Embed color (decimal int) |
| `sync_commands` | Register slash commands on startup |
| `[database]` | PostgreSQL connection |
| `[[discord_nodes]]` | Array of Node bots (at least one) |
| `[[lavalink_nodes]]` | Array of Lavalink nodes (at least one) |

### 5. Build and run

```bash
make build   # builds ./bot with embedded git hash + build stamp
make run     # build + run
```

## Running under systemd

```bash
echo 'SERVICE_NAME := MusicBot.service' > config.make
make service-install
make service-enable
make service-start
make service-status
```

`service-install` builds the binary, substitutes paths into `template.service`, and registers the unit under `/etc/systemd/system/`.

## License

[GNU AGPL-3.0](LICENSE). If you run a modified version as a service (including as a Discord bot), the source code of your modifications must be made available to the users of that service.
