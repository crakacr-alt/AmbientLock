# Как выпускать новую версию

Этот файл нужен, чтобы релиз не зависел от памяти.

1. Выбрать номер версии по SemVer.
2. Обновить `Version` в `internal/app/app.go`.
3. Обновить версию в README и `CITATION.cff`.
4. Записать изменения в `CHANGELOG.md`.
5. Добавить короткие release notes в `docs/RELEASE_<version>.md`.
6. Запустить локально:

```bash
gofmt -w .
go vet ./...
go test -race ./...
go build ./cmd/ambient
```

7. Сделать отдельную ветку и pull request.
8. Дождаться зелёных CI и CodeQL.
9. Слить PR в `main`.
10. Создать tag вида `v0.1.2`.

После push тега workflow сам проверит совпадение версии, пересоберёт проект,
посчитает SHA256 и создаст GitHub Release.

Важно: tag не создаём до зелёного `main`.
