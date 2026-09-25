# Threat model

## Что мы хотим заметить

AmbientLock полезен, когда изменение кода незаметно расширило внешние зависимости
или возможности процесса:

- появился новый executable;
- добавилось чтение системного/проектного файла;
- появилась запись в новую область filesystem;
- добавился новый network endpoint;
- процессу стали передавать новые ENV names.

## Что находится вне threat model v0.1

- kernel compromise;
- rootkit, который подделывает syscall observation;
- anti-debugging против strace;
- операции, которые не входят в отслеживаемый syscall subset;
- доказательство того, что переданная ENV variable реально была прочитана;
- hostname attribution для уже resolved IP;
- prevention до выполнения syscall.

## Главная ценность

v0.1 — это change detector для development/CI, а не boundary безопасности.
Он уменьшает количество невидимых изменений и создаёт основу для будущего
preventive backend.
