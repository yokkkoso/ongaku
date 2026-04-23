# MusicBot

***English** · [Русский](README_ru.md)*

Multi-instance Discord music bot written in Go, powered by Lavalink.

---

## Features

- Playback from **Spotify**, **Yandex Music**, **SoundCloud**, **YouTube**, and any source supported by Lavalink (requires modifying [/play](./internal/handlers/commands/play.go) and [/search](./internal/handlers/commands/search.go) files).
- Track and album search via `/search`.
- Queue with repeat modes.
- Slash commands + interactive button-based player (pause, next, volume, repeat, stop).
- Auto-leave from an empty voice channel after 5 minutes.
- Queue recovery after restart.
- Support for [DAVE](https://github.com/thomas-vilte/dave-go).

## Architecture

The bot runs as two subsystems:

- **DJ** — the main bot that receives users' slash commands.
- **Node** — a pool of separate bots that actually connect to voice channels. The DJ picks a free Node per `(guild, channel)` pair and delegates playback to it through Lavalink.

This split allows simultaneous playback in multiple voice channels of the same guild — concurrency is bounded by the number of configured Node bots.

## Requirements

|                |                                                            |
|----------------|------------------------------------------------------------|
| **Go**         | 1.26+                                                      |
| **PostgreSQL** | 13+                                                        |
| **Lavalink**   | 4.x — https://lavalink.dev/                                |
| **Build**      | `CGO_ENABLED=1` + C/C++ toolchain (gcc/clang, make, cmake) |

## Configuration (config.toml)

| Field                | Purpose                                                                             |
|----------------------|-------------------------------------------------------------------------------------|
| `dj_token`           | DJ bot token                                                                        |
| `color`              | Embed color (decimal int)                                                           |
| `sync_commands`      | Whether to register slash commands on startup (Recomend to disable after first run) |
| `[database]`         | PostgreSQL connection                                                               |
| `[[discord_nodes]]`  | Array of Node bots (at least one)                                                   |
| `[[lavalink_nodes]]` | Array of Lavalink nodes (at least one)                                              |

## Running with Docker

The bot image is published to the GitLab Container Registry. Postgres and Lavalink must be set up separately (via your package manager, separate containers, etc.) and their addresses supplied in `config.toml`.

1. Download the example `config.toml` and fill it in:

   ```bash
   curl -o config.toml https://gitlab.com/yokkkoso/musicbot/-/raw/master/configs/config.example.toml
   ```

2. Run the container:

   ```bash
   docker run -d \
     --name musicbot \
     --restart unless-stopped \
     --network host \
     -v "./config.toml:/app/configs/config.toml:ro" \
     registry.gitlab.com/yokkkoso/musicbot:latest
   ```

`--network host` gives the container direct access to Postgres/Lavalink on `localhost`. If the services run on a different host, put their IP or DNS in `config.toml` and drop `--network host`.

## Running on Linux (systemd)

### 1. Clone

```bash
git clone https://gitlab.com/yokkkoso/musicbot musicbot
cd musicbot
```

### 2. Lavalink

Run your own Lavalink server following the [official documentation](https://lavalink.dev/) and install the [lavasrc](https://github.com/topi314/LavaSrc), [lavasearch](https://github.com/topi314/LavaSearch), and [youtube-plugin](https://github.com/lavalink-devs/youtube-source) plugins.

### 3. Configuration

```bash
cp configs/config.example.toml configs/config.toml
```

Fill with your values

### 4. Build and run

```bash
make build   # build
make run     # build + run
echo 'SERVICE_NAME := MusicBot.service' > config.make
make service-install
make service-enable
make service-start
make service-status
```

## License

[GNU AGPL-3.0](LICENSE). If you run a modified version as a service (including as a Discord bot), the source code of your modifications must be made available to the users of that service.

---

*The only thing AI-generated here is this README — I was too lazy to write it myself, though I ended up rewriting it anyway.*
