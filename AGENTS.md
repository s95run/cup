# Кубок С95 (cup) — AGENTS.md

## Платформа С95

Общие правила для всех репозиториев `github.com/s95run/*`. Блок одинаковый во всех репозиториях — меняйте синхронно.

### Принципы

- **TDD**: сначала тест на поведение, затем реализация. Go — table-driven тесты на стандартном `testing`; доменная логика (`internal/domain`) без I/O и полностью покрыта юнит-тестами.
- **DRY**, **KISS**, **YAGNI**, **SOLID**: минимальный diff; не менять HTTP-роутер (chi), драйвер БД (pgx), мигратор (goose) «заодно».
- Сложность — ориентир из s95.ru: функция ≤ ~16 строк логики, цикломатическая сложность ≤ 8, строка ≤ 124 символа.
- Внешние HTTP в тестах — только заглушки (`httptest.Server`; в s95.ru — WebMock/VCR). Реальный s95.ru из тестов не вызывать.
- Без N+1: к БД — батч-запросы, к s95.ru — батч-эндпоинты и rate limit клиента.
- Форматирование по `.editorconfig` s95.ru: UTF-8, LF, финальный перевод строки, 2 пробела (YAML/TS/Vue/HCL); Go — `gofmt`; Makefile — табы.

### Проверки

- Источник истины — **GitHub Actions**; в `main` мержим только с зелёными обязательными проверками.
- Перед push агент прогоняет локально то же, что в CI (см. «Команды» в AGENTS.md сервиса). Если локально невозможно (нет Docker, секретов, доступа к Nubes) — push и разбор `gh run view --log-failed`.
- Порядок job'ов как в s95.ru: security → lint → test → deploy; deploy только из `main` после зелёного test.
- При падении CI: разобрать лог, исправить код или workflow, снова push. Не ослаблять CI ради зелёной сборки: не отключать линтеры, не `t.Skip`, не `--no-verify`.
- Миграции — только новыми файлами goose; применённые миграции не редактировать.

### Окружения

- Облако **Nubes** (VMware Cloud Director). Инфраструктура — репозиторий `s95run/terraform`: `live/dev` (рабочее), `live/prod` (заготовка).
- **vm-app** — сервисы в Docker Compose за nginx, порты только на `127.0.0.1`.
- **vm-data** — общий PostgreSQL: у каждого сервиса своя схема и роль `<service>_app`; в чужие схемы не ходить. Bootstrap схемы — Ansible-роль `postgres_service_schema`.
- Деплой — Ansible (`deploy/ansible`), образы `ghcr.io/s95run/<service>-<component>`.
- Секреты в git не коммитить: на VM — `/etc/<service>/<service>.env`, в CI — GitHub Secrets; `terraform.tfvars`, `backend.hcl`, state — только локально или в S3.

### Стек (эталоны)

- **Новый бэкенд** — по образцу **cup**: Go 1.26, `cmd/<service>-api` + `internal/{config,domain,service,repository,transport/http}`, chi, pgx, goose, Viper (env `<SERVICE>_*`), healthcheck `/api/v1/healthz`, graceful shutdown, `go test -race ./...`.
- **Новый фронтенд** — по образцу **cup/web**: Vue 3 `<script setup lang="ts">`, TypeScript, Pinia, Vue Router, Vite, Tailwind CSS v4, Vitest; сборка `vue-tsc -b && vite build`; prod — nginx-образ.
- **Инфраструктура** — Terraform 1.12.2, `vmware/vcd` 3.14.2, модули в `modules/`, корни в `live/<env>`; проверки fmt/validate/test/TFLint/Trivy. Конфигурация VM — Ansible-роли.

Для новых сервисов — те же принципы, проверки и деплой; стек не подменять эталоном без необходимости.

### Интеграция с s95.ru

- s95.ru — Rails-монолит [vol1ura/Sat_9am_5km](https://github.com/vol1ura/Sat_9am_5km): Ruby 3.4.5, Rails 8.1, PostgreSQL, Sidekiq, Hotwire, ActiveAdmin, RSpec. Источник истины по атлетам, забегам и протоколам.
- Сервисы С95 не ходят в БД s95.ru — только HTTP API `/api/<service>/*` (Bearer API key) и SSO `/<service>_sso` (общий секрет).
- Изменения на стороне s95.ru — PR в Sat_9am_5km по его правилам: request spec + FactoryBot на каждый эндпоинт, WebMock/VCR для внешних HTTP; должны проходить `rubocop`, `brakeman`, `bundler-audit`, `database_consistency`, `yarn lint` (локально — `make checkup`). Черновик PR — в `contrib/sat_9am_5km_pr/` репозитория сервиса.
- Контракт API меняем обратно совместимо: сначала выкатка на s95.ru, затем в сервисе.

---

## Домен

**Кубок С95** (cup.s95.ru) — командный чемпионат: сетка этапов, заявки капитанов, протоколы с s95.ru и зачёт сезона. Вход через SSO s95.ru → JWT в cookie. Фоновый worker синхронизирует локации (раз в сутки) и протоколы (каждые 15 минут).

## Стек

Go (chi, pgx, goose, Viper, golang-jwt), Vue 3 + Vite + Tailwind v4, PostgreSQL (схема `cup`).

## Команды

- `make test` — `go test -race ./...` (для integration нужен Postgres со схемой `cup`, см. `deploy/postgres-init/`).
- `make lint` — `golangci-lint run ./...` (если установлен).
- `make build` — `bin/cup-api`.
- `make up` / `make down` / `make migrate` — локальный стек `deploy/docker-compose.yml` (использует `sudo`).
- `cd web && npm ci && npm run build && npm run test:unit` — как в CI.

### CI (`.github/workflows/ci.yml`)

- Job **backend**: Postgres 17, init схемы `cup`, `go test -race ./...`.
- Job **frontend**: `npm ci`, `npm run build`, `npm run test:unit`.
- **(цель)**: job security (`govulncheck`, audit npm), `golangci-lint` в CI, настоящий ESLint для `web`, сборка образов в ghcr с тегом git SHA вместо `:latest`.

## Конфиг

`configs/config.example.yaml` → `configs/config.yaml` или env `CUP_*` (см. README).

## Deploy

- Образы `ghcr.io/s95run/cup-api`, `ghcr.io/s95run/cup-web`; `deploy/docker-compose.prod.yml`, env — `/etc/cup/cup.env` на vm-app.
- Ansible: `deploy/ansible/playbooks/site.yml`, inventory `inventories/dev` (vm-app, vm-data).
- БД: схема `cup`, роль `cup_app` на vm-data.