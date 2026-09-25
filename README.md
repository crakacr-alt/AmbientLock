# AmbientLock

**AmbientLock** — экспериментальный open-source инструмент, который создаёт
`ambient.lock`: версионируемый контракт скрытых зависимостей обычной программы.

Идея простая: package manager знает зависимости проекта, но часто не знает, что
код во время реального запуска дополнительно читает `/etc/...`, вызывает
`/usr/bin/ffmpeg`, пишет в неожиданную папку, получает новые переменные окружения
или начинает подключаться к новому IP-адресу. AmbientLock делает такие изменения
видимыми в Git и CI.

> **Версия:** 0.1.0  
> **Статус:** рабочий Linux MVP, не sandbox и не система предотвращения атак.

## Зачем это нужно

Типичная проблема звучит так: **«у меня работает, а в CI/на сервере — нет»**.
Причина часто находится вне `package.json`, `requirements.txt` или `go.mod`:
локальный бинарник, системный файл, сокет, переменная окружения или сеть.

AmbientLock позволяет сначала **выучить** фактическое окружение команды, а затем
показывать изменения этого контракта в следующем запуске.

```text
source code + declared packages
            |
            v
      program execution
            |
       Linux strace
            |
            v
 files / exec / network / exposed env names
            |
            v
        ambient.lock
            |
      diff / enforce
```

## Что умеет v0.1.0

- запускает любую команду под `strace`;
- отслеживает успешные `execve`, `open/openat/creat`, `connect`;
- разделяет чтение и запись файлов;
- сохраняет используемые executable-файлы;
- сохраняет IPv4/IPv6/Unix endpoints, которые видит `connect(2)`;
- сохраняет **только имена** переменных окружения, переданных процессу;
- не записывает значения ENV, чтобы случайно не положить токен в Git;
- редактирует значения типовых секретных CLI-флагов в сохранённой команде;
- создаёт детерминированно отсортированный `ambient.lock`;
- показывает `diff` между baseline и новым запуском;
- в `enforce` возвращает ошибку, если программа получила новую capability;
- имеет unit tests, integration test и GitHub Actions CI.

## Установка

Требования для v0.1:

- Linux;
- Go 1.23+ для сборки из исходников;
- `strace`.

Ubuntu/Debian:

```bash
sudo apt update
sudo apt install -y strace golang-go
```

Сборка:

```bash
git clone https://github.com/crakacr-alt/AmbientLock.git
cd AmbientLock
go build -o ambient ./cmd/ambient
```

## Быстрый старт

Создать контракт:

```bash
./ambient learn -- python3 app.py
```

Появится файл:

```text
ambient.lock
```

Показать сводку:

```bash
./ambient show
```

Проверить изменения:

```bash
./ambient diff -- python3 app.py
```

Для CI:

```bash
./ambient enforce -- python3 app.py
```

Если новый код внезапно начал использовать `curl` и подключаться к новому адресу,
вывод может выглядеть так:

```text
Ambient contract diff:
+ executable: /usr/bin/curl
+ network: ipv4 203.0.113.10:443
enforce failed: new ambient capabilities detected
```

`enforce` в этом случае завершится кодом `4`.

## Формат ambient.lock

Это JSON с обычным именем `ambient.lock`, поэтому его удобно читать человеку и
сравнивать Git-ом.

Пример сокращённо:

```json
{
  "schema": 1,
  "tool": "ambientlock",
  "tool_version": "0.1.0",
  "command": ["python3", "app.py"],
  "executables": ["/usr/bin/python3.12"],
  "filesystem": {
    "reads": ["./config.yaml"],
    "writes": ["./cache/result.json"]
  },
  "network": [
    {"family": "ipv4", "address": "203.0.113.10", "port": 443}
  ],
  "environment": {
    "exposed_names": ["HOME", "PATH", "DATABASE_URL"]
  }
}
```

## Важное ограничение `enforce`

В v0.1 `enforce` — **detective control**, а не preventive sandbox.

То есть программа сначала выполняется под наблюдением, затем AmbientLock проверяет
новые capabilities и возвращает ненулевой exit code. Системный вызов пока не
блокируется до выполнения.

Preventive enforcement через Landlock/seccomp запланирован отдельно. Это важно:
README специально не выдаёт текущий MVP за готовый sandbox.

## Что именно считается ENV-зависимостью

На syscall-уровне нельзя честно определить каждый вызов `getenv()` обычным
`strace`. Поэтому v0.1 называет поле `exposed_names`: это имена переменных,
которые были переданы через `execve`.

Значения не сохраняются никогда. По этой же причине новые ENV names видны в
`diff`, но **не ломают `enforce` по умолчанию**. Если для конкретного CI это
нужно, включите строгий режим:

```bash
./ambient enforce --strict-env -- python3 app.py
```

## Архитектура

```text
cmd/ambient              CLI entry point
internal/app             команды learn/diff/enforce/show
internal/trace           запуск strace + безопасный parser
internal/model           schema и чтение/запись ambient.lock
internal/contract        сравнение двух контрактов
```

Подробно: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Почему код написан просто

Проект специально оформлен так, чтобы его мог разобрать студент 1 курса:

- стандартная библиотека Go без тяжёлого framework;
- маленькие пакеты с одной ответственностью;
- комментарии объясняют не только **что**, но и **почему**;
- системные ограничения описаны честно;
- parser тестируется на фиксированных строках `strace`;
- интеграционный тест запускается только когда `strace` реально доступен.

Начать разбор кода лучше с [docs/STUDENT_GUIDE.md](docs/STUDENT_GUIDE.md).

## Проверка проекта

Одной командой:

```bash
make check
```

Или вручную:

```bash
gofmt -w .
go vet ./...
go test -race ./...
go build ./cmd/ambient
```

Tag `v*` запускает release workflow, который собирает Linux amd64/arm64 binaries и SHA256SUMS.

## Безопасность и приватность

AmbientLock не отправляет телеметрию и не требует облачного сервиса.

Но `ambient.lock` всё равно может содержать чувствительные **пути к файлам** или
сетевые адреса. Перед публикацией lock-файла его нужно просмотреть.

Подробнее: [SECURITY.md](SECURITY.md) и [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md).

## Roadmap

Следующие важные этапы:

1. DNS/hostname correlation, а не только resolved IP;
2. path normalization для контейнеров и workspace-relative paths;
3. optional ignore/policy file;
4. preventive Linux enforcement через Landlock/seccomp;
5. более точное наблюдение ENV-read без сохранения значений;
6. macOS backend;
7. Windows ETW backend;
8. PR-friendly SARIF/GitHub annotation output.

Полный план: [ROADMAP.md](ROADMAP.md).

## Научная и инженерная честность

AmbientLock не утверждает, что уже видит **все** зависимости процесса. v0.1
наблюдает конкретный набор Linux syscalls и создаёт полезный минимальный контракт.
Цель проекта — постепенно расширять coverage, не маскируя ограничения красивыми
маркетинговыми заявлениями.

## Лицензия

MIT. См. [LICENSE](LICENSE).
