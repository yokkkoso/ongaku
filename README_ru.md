# MusicBot

*[English](README.md) · **Русский***

Мульти-инстансный Discord-музыкальный бот на Go на основе Lavalink.

---

## Возможности

- Воспроизведение из **Spotify**, **Yandex Music**, **SoundCloud**, **YouTube** и любых источников, поддерживаемых Lavalink (Для этого нужно изменить [/play](./internal/handlers/commands/play.go) и [/search](./internal/handlers/commands/search.go) файлы).
- Поиск треков и альбомов через `/search`.
- Очередь с режимами повтора.
- Slash-команды + интерактивный плеер на кнопках (пауза, следующий, громкость, повтор, остановка).
- Автоматический выход из пустого голосового канала через 5 минут.
- Восстановление очередей после рестарта.
- Поддержка [DAVE](https://github.com/thomas-vilte/dave-go).

## Архитектура

Бот состоит из двух подсистем:

- **DJ** — основной бот, который принимает slash-команды пользователей.
- **Node** — набор отдельных ботов, которые и подключаются в голосовые каналы. DJ подбирает свободную ноду на пару `(guild, channel)` и делегирует ей воспроизведение через Lavalink.

Такая схема позволяет одновременно играть музыку в разных голосовых каналах одного сервера — количество параллельных воспроизведений ограничено числом настроенных Node-ботов.

## Требования

|                |                                                            |
|----------------|------------------------------------------------------------|
| **Go**         | 1.26+                                                      |
| **PostgreSQL** | 13+                                                        |
| **Lavalink**   | 4.x — https://lavalink.dev/                                |
| **Сборка**     | `CGO_ENABLED=1` + C/C++ toolchain (gcc/clang, make, cmake) |

## Конфигурация (config.toml)

| Поле                 | Назначение                                                                                 |
|----------------------|--------------------------------------------------------------------------------------------|
| `dj_token`           | токен DJ-бота                                                                              |
| `color`              | цвет embed-ов (decimal int)                                                                |
| `sync_commands`      | регистрировать ли slash-команды при старте (рекомендуется отключить после первого запуска) |
| `[database]`         | данные PostgreSQL                                                                          |
| `[[discord_nodes]]`  | массив Node-ботов (минимум один)                                                           |
| `[[lavalink_nodes]]` | массив Lavalink-нод (минимум один)                                                         |


## Запуск через Docker

Образ бота публикуется в GitHub Container Registry. Postgres и Lavalink нужно поднять самостоятельно (через пакетный менеджер, отдельные контейнеры и т.п.) и указать их адреса в `config.toml`.

1. Скачать пример `config.toml` и заполнить:

   ```bash
   curl -o config.toml https://raw.githubusercontent.com/yokkkoso/musicbot/master/configs/config.example.toml
   ```

2. Запустить контейнер:

   ```bash
   docker run -d \
     --name musicbot \
     --restart unless-stopped \
     --network host \
     -v "./config.toml:/app/configs/config.toml:ro" \
     ghcr.io/yokkkoso/musicbot:latest
   ```

`--network host` даёт контейнеру прямой доступ к Postgres/Lavalink на `localhost`. Если сервисы на другом хосте — указать их IP или DNS в `config.toml` и убрать `--network host`.

## Запуск на Linux (systemd)

### 1. Клонирование

```bash
git clone https://github.com/yokkkoso/musicbot musicbot
cd musicbot
```

### 2. Lavalink

Нужно поднять свой сервер согласно [официальной документации](https://lavalink.dev/) и установить плагины [lavasrc](https://github.com/topi314/LavaSrc), [lavasearch](https://github.com/topi314/LavaSearch), [youtube-plugin](https://github.com/lavalink-devs/youtube-source).

### 3. Конфигурация

```bash
cp configs/config.example.toml configs/config.toml
```

Заполнить своими значениями.

### 4. Сборка и запуск

```bash
make build   # сборка
make run     # сборка + запуск
echo 'SERVICE_NAME := MusicBot.service' > config.make
make service-install
make service-enable
make service-start
make service-status
```

`service-install` собирает бинарник, подставляет пути в `template.service` и регистрирует юнит в systemd.

## Лицензия

[GNU AGPL-3.0](LICENSE). Если вы поднимаете модифицированную версию как сервис (в том числе Discord-бота), исходный код модификаций должен быть доступен пользователям сервиса.

---

*Единственное что сгенерировано через AI - этот README, ибо мне было лень писать руками, хотя все равно я по итогу его переписал*
