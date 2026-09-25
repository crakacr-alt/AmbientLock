# Changelog

## 0.2.0 — 2026-09-25

Первый функциональный релиз после MVP.

### Added

- `.ambientignore` с правилами для file/read/write/exec/env/network;
- glob-поддержка `*`, `?` и `**`;
- `--ignore FILE` и `--no-ignore`;
- `--json-report FILE` для `diff` и `enforce`;
- JSON report автоматически создаёт родительские каталоги;
- машинный JSON с `changed`, `new_capabilities` и полным набором изменений.

### Changed

- ignore policy применяется и к baseline, и к новому запуску;
- неполные typed-правила вроде `read` без шаблона теперь отклоняются с ошибкой;
- schema `ambient.lock` остаётся совместимой: версия схемы не менялась.

## 0.1.2 — 2026-09-25

Patch-релиз инфраструктуры и процесса публикации.

### Added

- CodeQL-анализ Go-кода;
- Dependabot для Go modules и GitHub Actions;
- CODEOWNERS;
- отдельная инструкция по выпуску версий.

### Changed

- CI переведён на актуальные official GitHub Actions;
- release workflow теперь использует GitHub CLI вместо стороннего publish action;
- перед публикацией tag сверяется с версией в исходниках;
- релиз повторно запускает vet, race tests и build;
- README получил CI/CodeQL badges.

## 0.1.1 — 2026-09-25

Небольшой patch-релиз после первого полного аудита.

### Fixed

- относительные пути после `chdir()` и `openat(dirfd, ...)` теперь берутся из kernel-resolved пути `strace -yy`;
- типовые секретные CLI-флаги вроде `--token`, `--password`, `--api-key` и `--client-secret` редактируются перед записью команды в `ambient.lock`;
- workspace paths нормализуются в `./...`, чтобы lock меньше зависел от абсолютного пути checkout;
- новые ENV names по умолчанию показываются в diff, но не ломают enforce без `--strict-env`;
- исправлен packaging/.gitignore, из-за которого каталог `cmd/ambient` мог случайно не попасть в архив.

### Tests

- parser сверялся с реальным Linux `strace` для `execve`, `openat` и non-blocking `connect(...)=EINPROGRESS`;
- добавлены regression-тесты для resolved paths и редактирования секретных аргументов;
- GitHub Actions проходит format, vet, race tests, build и CLI smoke test.

## 0.1.0 — 2026-09-25

Первый рабочий Linux MVP.

### Added

- `ambient learn`
- `ambient diff`
- `ambient enforce`
- `ambient show`
- strace backend для exec/file/network наблюдения
- `ambient.lock` schema v1
- ENV names without values
- unit tests и Linux integration test
- GitHub Actions CI
- русская учебная документация
