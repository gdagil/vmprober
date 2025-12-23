# Troubleshooting

Руководство по решению распространенных проблем VMProber.

## Общие проблемы

### Приложение не запускается

#### Проблема: Ошибка загрузки конфигурации

**Симптомы:**
```
Failed to load configuration: failed to parse config: yaml: ...
```

**Решение:**
1. Проверьте синтаксис YAML файла:
   ```bash
   yamllint config.yaml
   ```
2. Проверьте валидность конфигурации
3. Убедитесь, что файл существует и доступен для чтения

#### Проблема: Порт уже занят

**Симптомы:**
```
Failed to start HTTP server: listen tcp :8429: bind: address already in use
```

**Решение:**
1. Измените порт в конфигурации:
   ```yaml
   listen:
     port: 8081
   ```
2. Или остановите процесс, использующий порт:
   ```bash
   lsof -i :8429
   kill <PID>
   ```

### Метрики не экспортируются

#### Проблема: Endpoint `/metrics` не отвечает

**Симптомы:**
```
curl http://localhost:8429/metrics
# Пустой ответ или ошибка
```

**Решение:**
1. Проверьте, что HTTP сервер запущен:
   ```bash
   curl http://localhost:8429/health
   ```
2. Проверьте логи на наличие ошибок
3. Убедитесь, что `pull.enabled: true` в конфигурации

#### Проблема: Метрики пустые

**Симптомы:**
Метрики экспортируются, но все значения равны 0.

**Решение:**
1. Проверьте, что цели настроены в конфигурации
2. Проверьте логи на наличие ошибок выполнения проб
3. Убедитесь, что цели доступны и порты открыты

### Пробы не выполняются

#### Проблема: Все пробы завершаются ошибкой

**Симптомы:**
```
vmprober_probe_failure_total увеличивается
vmprober_probe_success_total остается 0
```

**Решение:**
1. Проверьте доступность целей:
   ```bash
   telnet example.com 80
   ```
2. Проверьте firewall правила
3. Увеличьте timeout в конфигурации:
   ```yaml
   scheduler:
     timeouts:
       tcp: 10s
   ```
4. Включите debug логирование для детальной информации:
   ```yaml
   logging:
     level: "debug"
   ```

#### Проблема: ICMP пробы не работают

**Симптомы:**
ICMP пробы всегда завершаются ошибкой.

**Решение:**
1. Проверьте права доступа (требуются root/администратор)
2. Проверьте настройки системы:
   ```bash
   # Linux
   sysctl net.ipv4.ping_group_range
   ```
3. Используйте альтернативную библиотеку в конфигурации:
   ```yaml
   probes:
     icmp:
       library: "gopacket"
   ```

### Проблемы с производительностью

#### Проблема: Высокая загрузка CPU

**Симптомы:**
Высокое использование CPU процессом vmprober.

**Решение:**
1. Уменьшите количество одновременных проб:
   ```yaml
   scheduler:
     concurrent: 50
   ```
2. Увеличьте интервалы между пробами
3. Используйте jitter для распределения нагрузки:
   ```yaml
   scheduler:
     jitter: 0.2
   ```

#### Проблема: Высокое использование памяти

**Симптомы:**
Высокое использование памяти процессом vmprober.

**Решение:**
1. Уменьшите размер очереди:
   ```yaml
   scheduler:
     queue_size: 5000
   ```
2. Настройте WAL для ограничения размера:
   ```yaml
   wal:
     max_size: "500MB"
   ```
3. Проверьте на утечки памяти с помощью pprof

### Проблемы с WAL

#### Проблема: Ошибки записи в WAL

**Симптомы:**
```
Failed to write to WAL: permission denied
```

**Решение:**
1. Проверьте права доступа к директории WAL:
   ```bash
   ls -la /var/lib/vmprober/wal
   ```
2. Убедитесь, что директория существует
3. Измените путь WAL в конфигурации на доступную директорию

#### Проблема: WAL занимает много места

**Симптомы:**
Директория WAL растет без ограничений.

**Решение:**
1. Настройте retention:
   ```yaml
   wal:
     retention: 7d
     max_age: 168h
   ```
2. Настройте максимальный размер:
   ```yaml
   wal:
     max_size: "1GB"
   ```
3. Включите сжатие:
   ```yaml
   wal:
     compression: "gzip"
   ```

### Проблемы с push режимом

#### Проблема: Метрики не отправляются в VictoriaMetrics

**Симптомы:**
Метрики не появляются в VictoriaMetrics.

**Решение:**
1. Проверьте доступность endpoint:
   ```bash
   curl -X POST http://victoria-metrics:8428/api/v1/import/prometheus \
     -H "Content-Type: text/plain" \
     --data-binary "test_metric 1"
   ```
2. Проверьте аутентификацию (токены, credentials)
3. Включите debug логирование для детальной информации
4. Проверьте retry настройки

#### Проблема: Ошибки аутентификации

**Симптомы:**
```
Failed to push metrics: 401 Unauthorized
```

**Решение:**
1. Проверьте токен/credentials в конфигурации
2. Убедитесь, что токен не истек
3. Проверьте формат аутентификации:
   ```yaml
   push:
     endpoints:
       - url: "..."
         auth:
           type: "bearer"
           token: "${VM_TOKEN}"
   ```

## Отладка

### Включение debug логирования

```yaml
logging:
  level: "debug"
  format: "json"
```

Или через командную строку:

```bash
./bin/vmprober --config=config.yaml --log-level=debug
```

### Использование pprof

Включите pprof в конфигурации:

```yaml
observability:
  pprof:
    enabled: true
    port: 6060
    host: "127.0.0.1"
```

Профилирование:

```bash
# CPU профиль
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory профиль
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine профиль
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### Проверка конфигурации

```bash
# Проверка синтаксиса YAML
yamllint config.yaml

# Валидация конфигурации (если поддерживается)
./bin/vmprober --config=config.yaml --validate
```

### Проверка сетевых соединений

```bash
# Проверка доступности цели
telnet example.com 80

# Проверка DNS
nslookup example.com

# Проверка маршрутизации
traceroute example.com
```

## Логи

### Просмотр логов

**Docker:**
```bash
docker logs vmprober
docker logs -f vmprober  # Следить за логами
```

**Kubernetes:**
```bash
kubectl logs -f deployment/vmprober
```

**Systemd:**
```bash
sudo journalctl -u vmprober -f
```

### Поиск в логах

```bash
# Поиск ошибок
docker logs vmprober 2>&1 | grep -i error

# Поиск по времени
docker logs vmprober --since 1h

# Поиск конкретной цели
docker logs vmprober 2>&1 | grep "example.com"
```

## Мониторинг

### Проверка метрик

```bash
# Все метрики
curl http://localhost:8429/metrics

# Только метрики VMProber
curl http://localhost:8429/metrics | grep vmprober

# Конкретная метрика
curl http://localhost:8429/metrics | grep vmprober_probe_success_total
```

### Проверка здоровья

```bash
# Health check
curl http://localhost:8429/health

# Readiness check
curl http://localhost:8429/ready
```

### Prometheus запросы для диагностики

```promql
# Количество активных целей
count(vmprober_probe_attempts_total)

# Failure rate
rate(vmprober_probe_failure_total[5m]) / rate(vmprober_probe_attempts_total[5m])

# Средний RTT
rate(vmprober_probe_rtt_seconds_sum[5m]) / rate(vmprober_probe_rtt_seconds_count[5m])
```

## Получение помощи

Если проблема не решена:

1. Соберите информацию:
   - Версия VMProber
   - Конфигурация (без секретов)
   - Логи с debug уровнем
   - Метрики (`/metrics`)
   - Информация об окружении (OS, версия Go)

2. Создайте issue в репозитории с:
   - Описанием проблемы
   - Шагами для воспроизведения
   - Собранной информацией

## См. также

- [Примеры использования](examples.md)
- [Конфигурация](configuration.md)
- [Развертывание](deployment.md)

