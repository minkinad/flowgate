# Инструкции для агентов

- README и `docs/ARCHITECTURE.md` — контракт текущей реализации. Не объявляйте planned модули готовыми.
- Новые CLI команды — `cmd/flowgate`, parsing/validation — `internal/config`, matcher — `internal/routing`, HTTP lifecycle/proxy — `internal/gateway`.
- Направление зависимостей: cmd → gateway/routing/config; gateway → routing/config; routing → config. Не импортируйте cmd из internal.
- Используйте `context.Context`, `slog`, явные зависимости; не создавайте глобальные mutable переменные. `gofmt`, таблицы тестов для маршрутов и `httptest` для proxy.
- Не логируйте тела и секреты. Не доверяйте forwarding headers, не публикуйте admin endpoints без сетевой защиты. Таймауты обязательны.
- Проверка: `make setup`, `make check`, `make build`, `make docker-build` при доступном Docker.
- Definition of Done: сценарий реально работает, тест на граничный случай добавлен, README/roadmap обновлены, `go vet` и race detector проходят.
