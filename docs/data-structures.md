# VMProber Data Structures

## Обзор структур данных

VMProber использует строго типизированные структуры данных для обеспечения типобезопасности и производительности. Все структуры спроектированы для минимального использования памяти и максимальной производительности.

## Основные типы

### ProbeType
```go
type ProbeType string

const (
    ProbeTypeTCP  ProbeType = "tcp"
    ProbeTypeUDP  ProbeType = "udp"
    ProbeTypeICMP ProbeType = "icmp"
)
```

### NetworkFamily
```go
type NetworkFamily string

const (
    NetworkFamilyInet  NetworkFamily = "inet"  // IPv4
    NetworkFamilyInet6 NetworkFamily = "inet6" // IPv6
    NetworkFamilyAny   NetworkFamily = "any"   // Любое
)
```

## Core Data Structures

### ProbeResult
Представляет результат выполнения пробы с детальной информацией о производительности и ошибках.

```go
type ProbeResult struct {
    Success      bool          // Успешность пробы
    RTT          time.Duration // Время отклика
    Error        string        // Описание ошибки (если есть)
    Attempt      int           // Номер попытки
    Timestamp    time.Time     // Время выполнения
    SourceIP     string        // Исходный IP адрес
    TargetIP     string        // Целевой IP адрес
    TargetPort   int           // Целевой порт
    TLS          bool          // Использовался ли TLS
    Protocol     ProbeType     // Протокол пробы
    Role         string        // Роль (client/server)
    SocketFamily string        // Семейство сокетов
    Payload      []byte        // Отправленные данные
    Response     []byte        // Полученные данные
    DNSResult    *DNSResult    // Результат DNS разрешения
}
```

### DNSResult
Результат DNS разрешения для оптимизации повторных запросов.

```go
type DNSResult struct {
    ResolvedIPs []string       // Разрешенные IP адреса
    LookupTime  time.Duration  // Время DNS запроса
    TTL         time.Duration  // Время жизни записи
    Error       string         // Ошибка DNS (если есть)
}
```

### Target
Определяет цель для мониторинга с полной конфигурацией.

```go
type Target struct {
    ID            string            // Уникальный идентификатор
    Host          string            // Хост или IP адрес
    Port          int               // Порт (для TCP/UDP)
    Protocol      ProbeType         // Тип пробы
    Interval      time.Duration     // Интервал между пробами
    Timeout       time.Duration     // Таймаут пробы
    Count         int               // Количество попыток
    Labels        map[string]string // Метки для группировки
    NetworkFamily NetworkFamily     // Семейство сетевых адресов
    TLS           *TLSConfig        // Конфигурация TLS
    UDP           *UDPConfig        // Конфигурация UDP
    ICMP          *ICMPConfig       // Конфигурация ICMP
    Enabled       bool              // Включена ли цель
    Priority      int               // Приоритет выполнения
    CreatedAt     time.Time         // Время создания
    UpdatedAt     time.Time         // Время обновления
}
```

### TLSConfig
Конфигурация TLS для безопасных TCP соединений.

```go
type TLSConfig struct {
    Enabled            bool     // Включен ли TLS
    InsecureSkipVerify bool     // Пропуск проверки сертификатов
    ServerName         string   // SNI имя сервера
    MinVersion         string   // Минимальная версия TLS
    MaxVersion         string   // Максимальная версия TLS
    CipherSuites       []string // Разрешенные шифры
    RootCAs            string   // Путь к корневым сертификатам
    ClientCert         string   // Путь к клиентскому сертификату
    ClientKey          string   // Путь к ключу клиента
}
```

### UDPConfig
Конфигурация UDP проб с поддержкой различных типов payload.

```go
type UDPConfig struct {
    PayloadType     string        // Тип payload (echo, random)
    PayloadSize     int           // Размер payload
    ResponseTimeout time.Duration // Таймаут ответа
    MaxPacketSize   int           // Максимальный размер пакета
    BindAddress     string        // Адрес для привязки
}
```

### ICMPConfig
Конфигурация ICMP проб с выбором библиотеки.

```go
type ICMPConfig struct {
    Library       string // Библиотека (systicmp, gopacket)
    SequenceStart int    // Начальный номер последовательности
    TTL           int    // Time To Live
    Data          []byte // Дополнительные данные
}
```

## Job Management

### Job
Представляет задачу для выполнения пробы в планировщике.

```go
type Job struct {
    ID          string        // Уникальный идентификатор задачи
    Target      Target        // Целевая конфигурация
    NextRun     time.Time     // Время следующего выполнения
    Interval    time.Duration // Интервал между выполнениями
    Jitter      float64       // Случайное отклонение (0.0-1.0)
    RetryCount  int           // Количество выполненных ретраев
    MaxRetries  int           // Максимальное количество ретраев
    Priority    int           // Приоритет задачи
    CreatedAt   time.Time     // Время создания
    Attempt     int           // Текущая попытка выполнения
}
```

## Event Processing

### NormalizedEvent
Унифицированное событие после нормализации результатов проб.

```go
type NormalizedEvent struct {
    Timestamp   time.Time              // Время события
    SeriesID    string                 // Идентификатор серии метрик
    Metrics     map[string]float64     // Метрики события
    Labels      map[string]string      // Метки для группировки
    Tags        []string               // Теги для дополнительной классификации
    Metadata    map[string]interface{} // Дополнительные метаданные
    Source      string                 // Источник события
}
```

### Metric
Представляет метрику в формате Prometheus.

```go
type Metric struct {
    Name        string            // Имя метрики
    Value       float64           // Значение метрики
    Timestamp   time.Time         // Время измерения
    Labels      map[string]string // Метки метрики
    Type        MetricType        // Тип метрики
    Help        string            // Описание метрики
    Buckets     []float64         // Bucket'ы для histogram
    Sum         float64           // Сумма для histogram/summary
    Count       uint64            // Количество наблюдений
    Quantiles   map[float64]float64 // Квантили для summary
}
```

### MetricType
```go
type MetricType string

const (
    MetricTypeGauge     MetricType = "gauge"
    MetricTypeCounter   MetricType = "counter"
    MetricTypeHistogram MetricType = "histogram"
    MetricTypeSummary   MetricType = "summary"
)
```

## Storage Layer

### Record
Запись в системе хранения (WAL или постоянное хранилище).

```go
type Record struct {
    ID          string                 // Уникальный идентификатор
    Timestamp   time.Time              // Время записи
    Type        string                 // Тип записи
    Data        map[string]interface{} // Данные записи
    Labels      map[string]string      // Метки записи
    SeriesID    string                 // Идентификатор серии
    Compression string                 // Тип компрессии
    Size        int64                  // Размер записи
}
```

## Configuration Management

### ConfigUpdate
Событие обновления конфигурации с детальной информацией об изменениях.

```go
type ConfigUpdate struct {
    Type        UpdateType     // Тип обновления
    OldConfig   *Config        // Предыдущая конфигурация
    NewConfig   *Config        // Новая конфигурация
    Timestamp   time.Time      // Время обновления
    Source      string         // Источник обновления
    Changes     []ConfigChange // Список изменений
}
```

### ConfigChange
Отдельное изменение в конфигурации.

```go
type ConfigChange struct {
    Path    string      // Путь к измененному полю
    OldValue interface{} // Предыдущее значение
    NewValue interface{} // Новое значение
    Op      ChangeOp    // Тип операции
}
```

### UpdateType
```go
type UpdateType string

const (
    UpdateTypeFull    UpdateType = "full"    // Полное обновление
    UpdateTypePartial UpdateType = "partial" // Частичное обновление
    UpdateTypeError   UpdateType = "error"   // Ошибка обновления
)
```

### ChangeOp
```go
type ChangeOp string

const (
    ChangeOpAdd    ChangeOp = "add"    // Добавление
    ChangeOpUpdate ChangeOp = "update" // Обновление
    ChangeOpDelete ChangeOp = "delete" // Удаление
)
```

## Health Monitoring

### HealthStatus
Общий статус здоровья системы.

```go
type HealthStatus struct {
    Status      string                 // Общий статус
    Timestamp   time.Time              // Время проверки
    Uptime      time.Duration          // Время работы системы
    Version     string                 // Версия приложения
    Checks      map[string]HealthCheck // Проверки компонентов
    Metadata    map[string]interface{} // Дополнительные метаданные
}
```

### HealthCheck
Проверка здоровья отдельного компонента.

```go
type HealthCheck struct {
    Status    string        // Статус проверки
    Message   string        // Сообщение о состоянии
    Timestamp time.Time     // Время проверки
    Duration  time.Duration // Время выполнения проверки
    Details   interface{}   // Детальная информация
}
```

### ReadyStatus
Статус готовности системы к работе.

```go
type ReadyStatus struct {
    Ready      bool                   // Готовность системы
    Timestamp  time.Time              // Время проверки
    Components map[string]ReadyCheck  // Проверки компонентов
    Metadata   map[string]interface{} // Дополнительные метаданные
}
```

### ReadyCheck
Проверка готовности компонента.

```go
type ReadyCheck struct {
    Ready     bool        // Готовность компонента
    Message   string      // Сообщение о готовности
    Timestamp time.Time   // Время проверки
    Details   interface{} // Детальная информация
}
```

## Performance Monitoring

### RateLimitInfo
Информация о rate limiting.

```go
type RateLimitInfo struct {
    Key        string        // Ключ ограничения
    Rate       float64       // Лимит запросов в секунду
    Burst      int           // Размер burst
    Remaining  int           // Оставшиеся запросы
    ResetTime  time.Time     // Время сброса лимита
    RetryAfter time.Duration // Время до следующей попытки
}
```

### WorkerInfo
Информация о воркере в пуле.

```go
type WorkerInfo struct {
    ID         string        // Идентификатор воркера
    Status     WorkerStatus  // Статус воркера
    CurrentJob string        // Текущая задача
    StartTime  time.Time     // Время запуска
    JobsDone   int64         // Выполненные задачи
    JobsFailed int64         // Неудачные задачи
    AvgTime    time.Duration // Среднее время выполнения
    Memory     int64         // Использование памяти
    CPU        float64       // Использование CPU
}
```

### WorkerStatus
```go
type WorkerStatus string

const (
    WorkerStatusIdle    WorkerStatus = "idle"    // Простаивает
    WorkerStatusRunning WorkerStatus = "running" // Выполняет задачу
    WorkerStatusError   WorkerStatus = "error"   // Ошибка
    WorkerStatusStopped WorkerStatus = "stopped" // Остановлен
)
```

## Statistics

### ProbeStats
Статистика выполнения проб.

```go
type ProbeStats struct {
    TotalProbes     int64         // Общее количество проб
    SuccessfulProbes int64        // Успешные пробы
    FailedProbes    int64         // Неудачные пробы
    AvgRTT          time.Duration // Среднее время отклика
    MinRTT          time.Duration // Минимальное время отклика
    MaxRTT          time.Duration // Максимальное время отклика
    SuccessRate     float64       // Процент успешных проб
    CurrentRPS      float64       // Текущий RPS
    PeakRPS         float64       // Пиковый RPS
}
```

### SystemStats
Системная статистика производительности.

```go
type SystemStats struct {
    Timestamp    time.Time       // Время сбора статистики
    Uptime       time.Duration   // Время работы системы
    Memory       MemoryStats     // Статистика памяти
    CPU          CPUStats        // Статистика CPU
    Network      NetworkStats    // Сетевая статистика
    Disk         DiskStats       // Дисковая статистика
    Goroutines   int             // Количество горутин
    GC           GCStats         // Статистика сборщика мусора
    Connections  ConnectionStats // Статистика соединений
}
```

### MemoryStats
Детальная статистика использования памяти.

```go
type MemoryStats struct {
    Alloc       uint64 // Выделенная память
    TotalAlloc  uint64 // Общая выделенная память
    Sys         uint64 // Память системы
    Lookups     uint64 // Поиски в памяти
    Mallocs     uint64 // Выделения
    Frees       uint64 // Освобождения
    HeapAlloc   uint64 // Память кучи
    HeapSys     uint64 // Системная память кучи
    HeapIdle    uint64 // Свободная память кучи
    HeapInuse   uint64 // Используемая память кучи
    HeapReleased uint64 // Освобожденная память кучи
    StackInuse  uint64 // Используемая память стека
    StackSys    uint64 // Системная память стека
    MSpanInuse  uint64 // Используемая память mspan
    MSpanSys    uint64 // Системная память mspan
    MCacheInuse uint64 // Используемая память mcache
    MCacheSys   uint64 // Системная память mcache
    BuckHashSys uint64 // Память buck hash
    GCSys       uint64 // Память GC
    OtherSys    uint64 // Другая системная память
}
```

### CPUStats
Статистика использования CPU.

```go
type CPUStats struct {
    User    float64 // Пользовательское время
    System  float64 // Системное время
    Idle    float64 // Время простоя
    Nice    float64 // Время nice
    Iowait  float64 // Время ожидания I/O
    Irq     float64 // Время обработки прерываний
    Softirq float64 // Время softirq
    Steal   float64 // Время украденное другими VM
}
```

### NetworkStats
Сетевая статистика.

```go
type NetworkStats struct {
    BytesSent     uint64 // Отправленные байты
    BytesRecv     uint64 // Полученные байты
    PacketsSent   uint64 // Отправленные пакеты
    PacketsRecv   uint64 // Полученные пакеты
    ErrorsIn      uint64 // Ошибки входящих пакетов
    ErrorsOut     uint64 // Ошибки исходящих пакетов
    DropIn        uint64 // Отброшенные входящие пакеты
    DropOut       uint64 // Отброшенные исходящие пакеты
}
```

### DiskStats
Дисковая статистика.

```go
type DiskStats struct {
    ReadBytes    uint64         // Прочитанные байты
    WriteBytes   uint64         // Записанные байты
    ReadCount    uint64         // Количество операций чтения
    WriteCount   uint64         // Количество операций записи
    ReadTime     time.Duration  // Время чтения
    WriteTime    time.Duration  // Время записи
    IOInProgress int64          // Операции I/O в процессе
    IOTime       time.Duration  // Общее время I/O
}
```

### GCStats
Статистика сборщика мусора.

```go
type GCStats struct {
    NumGC         uint32        // Количество циклов GC
    PauseTotal    time.Duration // Общее время пауз
    PauseAvg      time.Duration // Среднее время паузы
    PauseMin      time.Duration // Минимальное время паузы
    PauseMax      time.Duration // Максимальное время паузы
    LastGC        time.Time     // Время последнего GC
    GCCPUFraction float64       // Доля CPU на GC
}
```

### ConnectionStats
Статистика соединений.

```go
type ConnectionStats struct {
    Active   int // Активные соединения
    Idle     int // Простаивающие соединения
    Waiting  int // Ожидающие соединения
    MaxTotal int // Максимальное общее количество
}
```

## Error Handling

### ErrorInfo
Детальная информация об ошибке для улучшенного логирования и мониторинга.

```go
type ErrorInfo struct {
    Code        string                 // Код ошибки
    Message     string                 // Сообщение об ошибке
    Cause       string                 // Причина ошибки
    StackTrace  string                 // Трассировка стека
    Timestamp   time.Time              // Время возникновения
    Context     map[string]interface{} // Контекст ошибки
    Recoverable bool                   // Можно ли восстановиться
    Retryable   bool                   // Можно ли повторить
    RetryAfter  *time.Duration         // Время до следующей попытки
}
```

## System Information

### VersionInfo
Информация о версии приложения.

```go
type VersionInfo struct {
    Version   string // Версия приложения
    Commit    string // Хеш коммита
    BuildDate string // Дата сборки
    GoVersion string // Версия Go
}
```

### ConfigHash
Хеш конфигурации для обнаружения изменений.

```go
type ConfigHash struct {
    Hash       string    // Хеш конфигурации
    Timestamp  time.Time // Время вычисления хеша
    Source     string    // Источник конфигурации
    Size       int64     // Размер конфигурации
}
```

## Event System

### EventType
```go
type EventType string

const (
    EventTypeProbeStart    EventType = "probe_start"
    EventTypeProbeComplete EventType = "probe_complete"
    EventTypeProbeError    EventType = "probe_error"
    EventTypeConfigReload  EventType = "config_reload"
    EventTypeSystemStart   EventType = "system_start"
    EventTypeSystemStop    EventType = "system_stop"
    EventTypeWorkerStart   EventType = "worker_start"
    EventTypeWorkerStop    EventType = "worker_stop"
    EventTypeMetricPushed  EventType = "metric_pushed"
    EventTypeWALWrite      EventType = "wal_write"
    EventTypeWALRead       EventType = "wal_read"
)
```

### Event
Событие системы для мониторинга и логирования.

```go
type Event struct {
    ID        string     // Уникальный идентификатор события
    Type      EventType  // Тип события
    Timestamp time.Time  // Время события
    Source    string     // Источник события
    Data      interface{} // Данные события
    Labels    map[string]string // Метки события
    Level     LogLevel   // Уровень логирования
}
```

### LogLevel
```go
type LogLevel string

const (
    LogLevelDebug LogLevel = "debug"
    LogLevelInfo  LogLevel = "info"
    LogLevelWarn  LogLevel = "warn"
    LogLevelError LogLevel = "error"
    LogLevelFatal LogLevel = "fatal"
)
```

## Принципы проектирования структур данных

### 1. Минимальное использование памяти
- Использование `time.Duration` вместо `int64` для времени
- Оптимизация размеров структур через `omitempty` теги
- Избегание избыточных полей

### 2. Производительность
- Предварительное выделение слайсов и карт где возможно
- Использование указателей только при необходимости
- Оптимизация для сериализации/десериализации

### 3. Типобезопасность
- Использование конкретных типов вместо `interface{}`
- Валидация на уровне структур
- Предотвращение паник через проверки

### 4. Расширяемость
- Добавление новых полей без breaking changes
- Использование метаданных для дополнительной информации
- Поддержка backward compatibility

### 5. Наблюдаемость
- Включение временных меток во все структуры
- Добавление идентификаторов для трассировки
- Поддержка метрик и статистики

### 6. Сериализация
- JSON теги для всех экспортируемых полей
- Поддержка как JSON, так и бинарной сериализации
- Оптимизация для сетевой передачи