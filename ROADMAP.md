# Roadmap AmbientLock

## v0.1 — Linux MVP

- [x] `learn`
- [x] `diff`
- [x] `enforce` в detective-режиме
- [x] `show`
- [x] files / executables / network endpoints
- [x] exposed ENV names без значений
- [x] unit + integration tests
- [x] CI

## v0.2 — удобство реального проекта

- [ ] `.ambientignore`
- [ ] workspace-relative path mode
- [ ] `ambient explain <resource>`
- [ ] JSON/SARIF report для CI
- [ ] DNS correlation: IP -> hostname, когда это можно наблюдать надёжно
- [ ] golden integration fixtures для Python/Node/Go

## v0.3 — более точное наблюдение

- [ ] eBPF backend как optional Linux tracer
- [ ] file descriptor lifecycle
- [x] resolved relative paths for successful open/openat via `strace -yy`
- [ ] subprocess tree visualization
- [ ] различать connect attempt и established connection точнее

## v0.4 — preventive enforcement

- [ ] Landlock policy для filesystem
- [ ] seccomp policy для выбранных syscall classes
- [ ] режим `ambient guard -- command`
- [ ] dry-run и понятные сообщения о блокировке

## v0.5+

- [ ] macOS backend
- [ ] Windows ETW backend
- [ ] package-manager correlation
- [ ] SBOM enrichment
- [ ] GitHub Action
- [ ] подписанные lock-файлы / provenance
