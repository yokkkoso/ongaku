# MusicBot

*[English](README.md) · **Русский***

Мульти-инстансный Discord-музыкальный бот на Go с поддержкой Spotify, Yandex Music, SoundCloud, YouTube и DAVE.

---

## Возможности

- Воспроизведение из **Spotify**, **Yandex Music**, **SoundCloud**, **YouTube** и любых источников, поддерживаемых Lavalink.
- Поиск треков и альбомов через `/search`, включение по кнопке из выдачи.
- Очередь с режимами повтора `off / track / queue`, пагинацией и покомпонентным удалением треков.
- Slash-команды + интерактивный плеер на кнопках (пауза, следующий, громкость, повтор, остановка).
- Автоматический выход из пустого голосового канала через 5 минут.
- Восстановление очередей после рестарта — состояние хранится в PostgreSQL.
- Поддержка [godave](https://github.com/disgoorg/godave).

## Архитектура

Бот состоит из двух ролей:

- **DJ** — единственный бот, который принимает slash-команды пользователей. Держит общий реестр нод и базу данных.
- **Node** — набор отдельных Discord-ботов, которые реально занимают голосовые каналы. DJ подбирает свободную ноду на пару `(guild, channel)` и делегирует ей воспроизведение через Lavalink.

Такая схема позволяет одновременно играть музыку в разных голосовых каналах одного сервера — количество параллельных воспроизведений ограничено числом настроенных Node-ботов.

## Требования

| | |
|---|---|
| **Go** | 1.26+ |
| **PostgreSQL** | 13+ |
| **Lavalink** | 4.x — https://lavalink.dev/ |
| **libdave** | собирается из исходников, см. ниже |
| **Сборка** | `CGO_ENABLED=1` + C/C++ toolchain (gcc/clang, make, cmake) |

## Установка

### 1. Клонирование

```bash
git clone <repo-url> musicbot
cd musicbot
```

### 2. libdave (обязательно перед сборкой)

Проект импортирует `github.com/disgoorg/godave/golibdave`, который динамически линкуется с нативной библиотекой **libdave** — реализацией Discord E2EE. Отдельного `go install` не существует: нужно запустить скрипт из репозитория [godave](https://github.com/disgoorg/godave), который скачает исходники libdave и соберёт `.so` / `.dylib` / `.dll`:

```bash
# Linux / macOS / WSL
./libdave_install.sh v1.1.0

# Windows
.\libdave_install.ps1 v1.1.0
```

### 3. Lavalink

Нужно поднять свой сервер согласно [официальной документации](https://lavalink.dev/) и плагины [lavasrc](https://github.com/topi314/LavaSrc) и [lavasearch](https://github.com/topi314/LavaSearch).

Минимально: Lavalink доступен на `host:port` с паролем, совпадающими с записью в `[[lavalink_nodes]]`.

### 4. Конфигурация

```bash
cp configs/config.example.toml configs/config.toml
```

Заполнить поля:

| Поле | Назначение |
|---|---|
| `dj_token` | токен DJ-бота |
| `color` | цвет embed-ов (decimal int) |
| `sync_commands` | регистрировать slash-команды при старте |
| `[database]` | доступ к PostgreSQL |
| `[[discord_nodes]]` | массив Node-ботов (минимум один) |
| `[[lavalink_nodes]]` | массив Lavalink-нод (минимум один) |

### 5. Сборка и запуск

```bash
make build   # собирает ./bot с вшитым git-hash + build stamp
make run     # сборка + запуск
```

## Запуск через systemd

```bash
echo 'SERVICE_NAME := MusicBot.service' > config.make
make service-install
make service-enable
make service-start
make service-status
```

`service-install` собирает бинарник, подставляет пути в `template.service` и регистрирует юнит в `/etc/systemd/system/`.

## Лицензия

[GNU AGPL-3.0](LICENSE). Если вы поднимаете модифицированную версию как сервис (в том числе Discord-бота), исходный код модификаций должен быть доступен пользователям сервиса.
