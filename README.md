# FlowGate [![CI](https://github.com/minkinad/flowgate/actions/workflows/ci.yml/badge.svg)](https://github.com/minkinad/flowgate/actions/workflows/ci.yml) [![CodeQL](https://github.com/minkinad/flowgate/actions/workflows/codeql.yml/badge.svg)](https://github.com/minkinad/flowgate/actions/workflows/codeql.yml) [![Security](https://github.com/minkinad/flowgate/actions/workflows/security.yml/badge.svg)](https://github.com/minkinad/flowgate/actions/workflows/security.yml)

FlowGate — экспериментальный HTTP API gateway на Go. Он проксирует запросы по host и префиксу пути, чтобы можно было постепенно вводить контролируемые политики трафика без смены входной точки сервисов.

**Статус:** рабочая основа v0.1, не готова как единственный production ingress. Лицензия Apache-2.0 планируется, файл LICENSE пока отсутствует.

## Реализовано

- HTTP reverse proxy на `net/http` и `httputil.ReverseProxy`; один upstream на маршрут.
- Выбор самого длинного совпадающего префикса, необязательный host, таймаут на маршрут.
- Строгая YAML-конфигурация с проверкой при запуске; окружение переопределяет адрес и настройки логов.
- `GET /health/live`, `GET /health/ready` (TCP доступность всех upstream), `GET /metrics` (формат Prometheus).
- JSON/text `slog`, request ID, graceful shutdown, self-check для контейнера.

**Запланировано:** балансировка, активные health checks, rate limiting, circuit breaker, retries, tracing, защищённый administrative API. Они пока не подключены.

## Быстрый старт

Нужен Go 1.22+. В другом терминале запустите локальный upstream, например `python3 -m http.server 9000`. Затем:

```bash
make setup
make run
curl http://127.0.0.1:8080/
curl http://127.0.0.1:8080/health/ready
curl http://127.0.0.1:8080/metrics
```

Проверить конфигурацию: `go run ./cmd/flowgate validate --config configs/flowgate.yaml`. Собрать бинарник: `make build`. Контейнер: `make docker-build`; образ использует `/configs/flowgate.yaml`, поэтому для другого upstream подмонтируйте свой файл. Docker Compose не требуется: обязательных внешних сервисов нет. Release workflow по тегу `v*` подготовлен для публикации multi-architecture образа в GHCR; в GitHub он ещё не выполнялся.

## Конфигурация

См. `configs/flowgate.yaml`. Поля `server.listen`, `server.log_format` (`json`/`text`), `server.log_level` (`debug`/`info`/`warn`/`error`), `routes[].name`, `match.host`, `match.path_prefix`, `upstreams[].url`, `timeout`. Переменные `FLOWGATE_LISTEN`, `FLOWGATE_LOG_FORMAT`, `FLOWGATE_LOG_LEVEL` переопределяют YAML; маршрут задаётся только в YAML. Значения по умолчанию: `:8080`, JSON, info, 5s. Не храните секреты в YAML; URL с userinfo отклоняется. Образ содержит пример upstream `127.0.0.1:9000`, который нужно заменить для своей сети.

## Архитектура и тестирование

`internal/config` проверяет вход, `internal/routing` выбирает маршрут, `internal/gateway` владеет HTTP-сервером, транспортами и метриками. Подробности — [архитектура](docs/ARCHITECTURE.md) и [roadmap](docs/ROADMAP.md).

```bash
go test ./...
go test -race ./...
go vet ./...
make check
```

## Ограничения

Readiness проверяет TCP подключение к каждому upstream, но не бизнес-готовность приложения. Метрики пока ограничены общими счётчиками. Административные пути расположены на том же listener и не имеют аутентификации: публикуйте их только через доверенную сеть. WebSocket, gRPC и динамический reload не проверены. Контейнерная сборка зависит от доступности базовых образов.
