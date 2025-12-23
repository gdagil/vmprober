# Docker Compose Setup для VMProber

Этот docker-compose файл запускает полный стек мониторинга с VictoriaMetrics и Grafana.

## Компоненты

- **vmstorage** - хранилище VictoriaMetrics (порт 8482)
- **vminsert** - компонент для записи метрик (порт 8480)
- **vmselect** - компонент для чтения метрик (порт 8481)
- **vmprober** - приложение для мониторинга доступности (порт 8429)
- **grafana** - веб-интерфейс для визуализации метрик (порт 3000)

## Быстрый старт

1. Запустите все сервисы:
```bash
docker-compose up -d
```

2. Проверьте статус:
```bash
docker-compose ps
```

3. Откройте Grafana:
   - URL: http://localhost:3000
   - Авторизация отключена (анонимный доступ с правами Admin)

4. Дашборд VMProber будет автоматически загружен

## Хранилище данных

Все данные хранятся в папке `./data`:
- `./data/vmstorage` - данные VictoriaMetrics
- `./data/vmprober/wal` - Write-Ahead Log для vmprober
- `./data/grafana` - данные Grafana (дашборды, настройки)

## Остановка

```bash
docker-compose down
```

Для удаления всех данных:
```bash
docker-compose down -v
rm -rf ./data
```

## Проверка работы

- VMProber UI: http://localhost:8429
- VMProber Metrics: http://localhost:8429/metrics
- VictoriaMetrics Insert: http://localhost:8480
- VictoriaMetrics Select: http://localhost:8481
- VictoriaMetrics Storage: http://localhost:8482
- Grafana: http://localhost:3000

## Конфигурация

Конфигурация vmprober находится в `config.docker.yaml`. Для изменения настроек отредактируйте этот файл и перезапустите контейнер:

```bash
docker-compose restart vmprober
```


