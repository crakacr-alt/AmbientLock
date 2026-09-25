# Гайд по коду для студента

## С чего начать

Читайте проект в таком порядке:

1. `cmd/ambient/main.go`
2. `internal/app/app.go`
3. `internal/trace/runner.go`
4. `internal/trace/parser.go`
5. `internal/model/lock.go`
6. `internal/contract/diff.go`

Так виден полный путь данных: команда -> strace -> parser -> lock -> diff.

## Какие темы Go здесь можно изучить

### packages

Код разбит по ответственности, а папка `internal` запрещает использовать эти
пакеты как внешний публичный API.

### interfaces через стандартные io.Writer/io.Reader

Parser принимает `io.Reader`, а CLI принимает `io.Writer`. Поэтому тесты могут
использовать `strings.Reader` и `bytes.Buffer`, не трогая реальный терминал.

### errors

Ошибки оборачиваются через `%w`. Это сохраняет исходную ошибку и добавляет
контекст: не просто «permission denied», а «open trace file: permission denied».

### sets через map

В Go нет встроенного `set`, поэтому множество строк удобно хранить как:

```go
map[string]struct{}
```

Пустая структура почти не занимает дополнительной памяти.

### table / integration testing

Unit-тест parser-а использует искусственные строки `strace`. Integration test
при наличии Linux + strace реально запускает `/usr/bin/cat` и проверяет, что
`/etc/hostname` был замечен.

## Что попробовать самостоятельно

Хорошая учебная задача для следующего коммита:

1. добавить `.ambientignore`;
2. написать parser правил `glob`;
3. покрыть его тестами;
4. применить ignore после наблюдения, но до сохранения lock-файла;
5. объяснить в README, почему ignore не должен скрывать данные молча в strict mode.
