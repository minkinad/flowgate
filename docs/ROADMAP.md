# Roadmap FlowGate

- [x] v0.1 foundation: конфигурация, route matching, один upstream, таймаут, health, базовые метрики, тесты.
- [ ] v0.1 hardening: полноценные метрики статусов/latency, отдельный admin listener, проверки proxy headers и streaming, benchmark throughput и allocations.
- [ ] v0.2 resilience: token bucket с инъекцией часов, распределённый лимит через Redis только при реальном сценарии, breaker state machine, безопасные retries.
- [ ] v0.2 upstreams: weighted round-robin, least-connections, passive/active health checks, отказоустойчивая readiness.
- [ ] v0.3: adaptive concurrency, измеренный benchmark report.
- [ ] v0.4: canary и sticky routing, атомарный dynamic reload.
- [ ] v0.5: gRPC proxy и административный API с аутентификацией, RBAC и аудитом.
