#!/usr/bin/env sh
set -eu

# Небольшая программа для ручной проверки AmbientLock.
# Она читает один системный файл и пишет локальный результат.
cat /etc/hostname >/dev/null
printf 'ambientlock demo\n' > ./ambient-demo-output.txt
