# AmbientLock 0.1.2

Это инфраструктурный patch-релиз.

Что изменилось:

- CI переведён на актуальные official GitHub Actions;
- добавлен CodeQL;
- добавлен Dependabot для Go и GitHub Actions;
- release workflow больше не зависит от стороннего publish action;
- перед релизом tag сверяется с версией в исходниках;
- release candidate обязательно проходит vet, race tests и build;
- добавлены CODEOWNERS и понятный release process;
- README получил статусы CI/CodeQL.

Функциональная модель 0.1.x не изменилась: это Linux MVP на `strace`, а
`enforce` остаётся detective control.
