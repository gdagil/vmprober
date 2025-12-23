# Руководство разработчика

Руководство по разработке и расширению VMProber.

## Требования

- Go 1.21 или выше
- Make
- Git

## Настройка окружения

### Клонирование репозитория

```bash
git clone https://github.com/gdagil/vmprober.git
cd vmprober
```

### Установка зависимостей

```bash
make deps
```

### Проверка установки

```bash
go version
make build
```

## Структура проекта

```
vmprober/
├── cmd/
│   └── vmprober/          # Главный файл приложения
│       └── main.go
├── internal/              # Внутренние пакеты
│   ├── config/           # Управление конфигурацией
│   ├── probe/            # Система проб (TCP, UDP)
│   ├── scheduler/         # Планировщик задач
│   ├── metrics/          # Система метрик Prometheus
│   ├── server/           # HTTP сервер
│   └── types/            # Базовые типы
├── pkg/                  # Публичные пакеты
│   └── interfaces/       # Публичные интерфейсы
├── docs/                 # Документация
├── config.yaml.example   # Пример конфигурации
├── Makefile              # Make команды
├── go.mod                # Go модули
└── README.md             # Основной README
```

## Сборка

### Локальная сборка

```bash
make build
```

Бинарный файл будет создан в `bin/vmprober`.

### С версией

```bash
VERSION=1.0.0 make build
```

### Кросс-компиляция

```bash
GOOS=linux GOARCH=amd64 make build
```

## Тестирование

### Запуск всех тестов

```bash
make test
```

### Запуск с покрытием

```bash
make test-coverage
```

Откройте `coverage.html` в браузере для просмотра покрытия.

### Запуск конкретного теста

```bash
go test -v ./internal/probe/...
```

### Запуск с race detector

```bash
go test -race ./...
```

## Форматирование и линтинг

### Форматирование кода

```bash
make fmt
```

### Линтинг

```bash
make lint
```

Требуется установленный `golangci-lint`:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Разработка новых функций

### Добавление нового типа пробы

1. Создайте реализацию интерфейса `Probe` в `internal/probe/`
2. Зарегистрируйте в `Factory` (`internal/probe/factory.go`)
3. Добавьте тесты
4. Обновите документацию

Пример:

```go
// internal/probe/custom.go
package probe

import (
    "context"
    "github.com/vmprober/vmprober/internal/types"
    "github.com/vmprober/vmprober/pkg/interfaces"
)

type CustomProbe struct {
    // ...
}

func (p *CustomProbe) Execute(ctx context.Context, target types.Target) (*types.ProbeResult, error) {
    // Реализация
}

func (p *CustomProbe) Type() types.ProbeType {
    return types.ProbeTypeCustom
}

func (p *CustomProbe) Validate(config interface{}) error {
    // Валидация
}

func (p *CustomProbe) Close() error {
    // Очистка ресурсов
}
```

### Добавление новой метрики

1. Добавьте метрику в `internal/metrics/collector.go`
2. Обновите метод `Record`
3. Добавьте тесты
4. Обновите документацию

### Добавление нового endpoint

1. Добавьте handler в `internal/server/server.go`
2. Зарегистрируйте в `NewServer`
3. Добавьте тесты
4. Обновите документацию API

## Отладка

### Локальный запуск

```bash
./bin/vmprober --config=config.yaml.example --log-level=debug
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

### Использование Delve

```bash
# Установка
go install github.com/go-delve/delve/cmd/dlv@latest

# Запуск с отладчиком
dlv debug ./cmd/vmprober -- --config=config.yaml.example
```

## Git workflow

### Создание ветки

```bash
git checkout -b feature/new-feature
```

### Коммиты

Используйте конвенциональные коммиты:

```
feat: add ICMP probe support
fix: resolve scheduler race condition
docs: update configuration documentation
test: add tests for UDP probe
refactor: simplify metrics collector
```

### Pull Request

1. Создайте ветку для функции
2. Внесите изменения
3. Добавьте тесты
4. Обновите документацию
5. Создайте Pull Request

## Код-стайл

### Именование

- Пакеты: lowercase, одно слово
- Функции: `ExportFunction` для публичных, `internalFunction` для приватных
- Переменные: `exportedVar` для публичных, `internalVar` для приватных
- Константы: `ExportedConstant`

### Комментарии

Все публичные функции, типы и переменные должны иметь комментарии:

```go
// Collector собирает и экспортирует метрики
type Collector struct {
    // ...
}

// NewCollector создает новый коллектор метрик
func NewCollector(namespace string) *Collector {
    // ...
}
```

### Обработка ошибок

Всегда проверяйте ошибки:

```go
result, err := someFunction()
if err != nil {
    return nil, fmt.Errorf("failed to execute: %w", err)
}
```

## Тестирование

### Unit тесты

Создавайте тесты для каждого модуля:

```go
// internal/probe/tcp_test.go
package probe

import (
    "testing"
    "time"
    "github.com/vmprober/vmprober/internal/types"
)

func TestTCPProbe_Execute(t *testing.T) {
    // Тест
}
```

### Integration тесты

Для интеграционных тестов используйте теги:

```go
// +build integration

package probe

import (
    "testing"
)

func TestTCPProbe_Integration(t *testing.T) {
    // Интеграционный тест
}
```

Запуск:

```bash
go test -tags=integration ./...
```

### Mocking

Используйте интерфейсы для мокирования:

```go
type MockProbe struct {
    ExecuteFunc func(ctx context.Context, target types.Target) (*types.ProbeResult, error)
}

func (m *MockProbe) Execute(ctx context.Context, target types.Target) (*types.ProbeResult, error) {
    return m.ExecuteFunc(ctx, target)
}
```

## Документация

### Обновление документации

При добавлении новых функций обновляйте:

1. `docs/` - соответствующую документацию
2. `README.md` - если нужно
3. `config.yaml.example` - если добавлены новые опции конфигурации

### Генерация документации

```bash
# Go doc
go doc ./internal/probe

# Godoc сервер
godoc -http=:6060
```

## Производительность

### Бенчмарки

Создавайте бенчмарки для критичных функций:

```go
func BenchmarkProbeExecute(b *testing.B) {
    // Бенчмарк
}
```

Запуск:

```bash
go test -bench=. ./...
```

### Профилирование

См. раздел "Отладка" выше.

## CI/CD

### GitHub Actions

Пример workflow:

```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - run: make deps
      - run: make test
      - run: make lint
```

## См. также

- [Структура проекта](project-structure.md)
- [Модульные интерфейсы](module-interfaces.md)
- [Архитектура](architecture.md)

