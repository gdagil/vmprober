# VMProber HTTP Server

## Обзор HTTP сервера

HTTP сервер VMProber предоставляет REST API endpoints для экспорта метрик, health checks и readiness probes. Сервер поддерживает как HTTP, так и HTTPS протоколы, а также включает middleware для логирования, метрик и безопасности.

## Архитектура HTTP сервера

```mermaid
graph TB
    subgraph "HTTP Server Core"
        ROUTER[HTTP Router]
        HANDLERS[Request Handlers]
        MIDDLEWARE[Middleware Chain]
        SERVER[HTTP Server]
    end
    
    subgraph "Endpoints"
        METRICS_ENDPOINT[/metrics]
        HEALTH_ENDPOINT[/health]
        READY_ENDPOINT[/ready]
        DEBUG_ENDPOINT[/debug]
        API_ENDPOINTS[/api/v1/*]
    end
    
    subgraph "Middleware Stack"
        CORS_MW[CORS Middleware]
        AUTH_MW[Auth Middleware]
        RATE_LIMIT_MW[Rate Limit Middleware]
        LOGGING_MW[Logging Middleware]
        METRICS_MW[Metrics Middleware]
        RECOVERY_MW[Recovery Middleware]
        TIMEOUT_MW[Timeout Middleware]
    end
    
    subgraph "Security Layer"
        TLS_HANDLER[TLS Handler]
        CERT_MANAGER[Certificate Manager]
        SECURITY_HEADERS[Security Headers]
        REQUEST_VALIDATOR[Request Validator]
    end
    
    subgraph "Monitoring"
        REQUEST_METRICS[Request Metrics]
        RESPONSE_METRICS[Response Metrics]
        ERROR_TRACKING[Error Tracking]
        PERFORMANCE_MONITOR[Performance Monitor]
    end
    
    subgraph "Configuration"
        SERVER_CONFIG[Server Config]
        TLS_CONFIG[TLS Config]
        MIDDLEWARE_CONFIG[Middleware Config]
        RATE_LIMIT_CONFIG[Rate Limit Config]
    end
    
    %% Request Flow
    CLIENT[Client] --> ROUTER
    ROUTER --> MIDDLEWARE
    
    %% Middleware Chain
    MIDDLEWARE --> CORS_MW
    CORS_MW --> AUTH_MW
    AUTH_MW --> RATE_LIMIT_MW
    RATE_LIMIT_MW --> LOGGING_MW
    LOGGING_MW --> METRICS_MW
    METRICS_MW --> RECOVERY_MW
    RECOVERY_MW --> TIMEOUT_MW
    TIMEOUT_MW --> HANDLERS
    
    %% Handler Routes
    HANDLERS --> METRICS_ENDPOINT
    HANDLERS --> HEALTH_ENDPOINT
    HANDLERS --> READY_ENDPOINT
    HANDLERS --> DEBUG_ENDPOINT
    HANDLERS --> API_ENDPOINTS
    
    %% Security Flow
    SERVER --> TLS_HANDLER
    TLS_HANDLER --> CERT_MANAGER
    TLS_HANDLER --> SECURITY_HEADERS
    TLS_HANDLER --> REQUEST_VALIDATOR
    
    %% Monitoring Flow
    MIDDLEWARE --> REQUEST_METRICS
    HANDLERS --> RESPONSE_METRICS
    HANDLERS --> ERROR_TRACKING
    HANDLERS --> PERFORMANCE_MONITOR
    
    %% Configuration Flow
    SERVER_CONFIG --> SERVER
    TLS_CONFIG --> TLS_HANDLER
    MIDDLEWARE_CONFIG --> MIDDLEWARE
    RATE_LIMIT_CONFIG --> RATE_LIMIT_MW
```

## Основные компоненты

### 1. HTTP Router
Центральный маршрутизатор для обработки HTTP запросов.

### 2. Request Handlers
Обработчики для каждого endpoint.

### 3. Middleware Chain
Цепочка middleware для обработки запросов.

### 4. Security Layer
Компоненты безопасности и TLS.

### 5. Monitoring
Система мониторинга и метрик.

## Интерфейсы

### HTTPServer Interface
```go
type HTTPServer interface {
    // Start запускает HTTP сервер
    Start(ctx context.Context) error
    
    // Stop останавливает HTTP сервер
    Stop(ctx context.Context) error
    
    // GetAddr возвращает адрес сервера
    GetAddr() string
    
    // GetStats возвращает статистику сервера
    GetStats() ServerStats
    
    // RegisterHandler регистрирует обработчик
    RegisterHandler(ctx context.Context, pattern string, handler http.Handler) error
    
    // Middleware регистрирует middleware
    RegisterMiddleware(ctx context.Context, middleware Middleware) error
}
```

### Middleware Interface
```go
type Middleware interface {
    // Handle выполняет middleware логику
    Handle(ctx context.Context, next http.Handler) http.Handler
    
    // Name возвращает имя middleware
    Name() string
    
    // Priority возвращает приоритет выполнения
    Priority() int
}
```

### Handler Interface
```go
type Handler interface {
    // Handle обрабатывает HTTP запрос
    Handle(ctx context.Context, w http.ResponseWriter, r *http.Request) error
    
    // Method возвращает HTTP метод
    Method() string
    
    // Pattern возвращает URL паттерн
    Pattern() string
    
    // Middleware возвращает middleware для обработчика
    Middleware() []Middleware
}
```

## Core Data Structures

### ServerConfig
```go
type ServerConfig struct {
    ListenAddr     string            `yaml:"listen_addr" json:"listen_addr"`
    ListenPort     int               `yaml:"listen_port" json:"listen_port"`
    ReadTimeout    time.Duration     `yaml:"read_timeout" json:"read_timeout"`
    WriteTimeout   time.Duration     `yaml:"write_timeout" json:"write_timeout"`
    IdleTimeout    time.Duration     `yaml:"idle_timeout" json:"idle_timeout"`
    MaxHeaderBytes int               `yaml:"max_header_bytes" json:"max_header_bytes"`
    
    // TLS Configuration
    TLSEnabled     bool              `yaml:"tls_enabled" json:"tls_enabled"`
    TLSCertFile    string            `yaml:"tls_cert_file" json:"tls_cert_file"`
    TLSKeyFile     string            `yaml:"tls_key_file" json:"tls_key_file"`
    TLSMinVersion  string            `yaml:"tls_min_version" json:"tls_min_version"`
    
    // CORS Configuration
    CORSEnabled    bool              `yaml:"cors_enabled" json:"cors_enabled"`
    CORSOrigins    []string          `yaml:"cors_origins" json:"cors_origins"`
    CORSMethods    []string          `yaml:"cors_methods" json:"cors_methods"`
    CORSHeaders    []string          `yaml:"cors_headers" json:"cors_headers"`
    
    // Rate Limiting
    RateLimitEnabled bool            `yaml:"rate_limit_enabled" json:"rate_limit_enabled"`
    RateLimitRPS     float64         `yaml:"rate_limit_rps" json:"rate_limit_rps"`
    RateLimitBurst   int             `yaml:"rate_limit_burst" json:"rate_limit_burst"`
    
    // Security
    AuthEnabled    bool              `yaml:"auth_enabled" json:"auth_enabled"`
    AuthType       string            `yaml:"auth_type" json:"auth_type"`
    AuthConfig     map[string]string `yaml:"auth_config" json:"auth_config"`
    
    // Middleware
    MiddlewareConfig map[string]interface{} `yaml:"middleware_config" json:"middleware_config"`
    
    // Health Check
    HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
    ReadinessCheckInterval time.Duration `yaml:"readiness_check_interval" json:"readiness_check_interval"`
}
```

### ServerStats
```go
type ServerStats struct {
    Uptime            time.Duration     `json:"uptime"`
    TotalRequests     int64             `json:"total_requests"`
    TotalErrors       int64             `json:"total_errors"`
    ActiveConnections int64             `json:"active_connections"`
    RequestRate       float64           `json:"request_rate"`
    ErrorRate         float64           `json:"error_rate"`
    AvgResponseTime   time.Duration     `json:"avg_response_time"`
    MemoryUsage       uint64            `json:"memory_usage"`
    CPUUsage          float64           `json:"cpu_usage"`
    LastRequest       time.Time         `json:"last_request"`
    EndpointStats     map[string]*EndpointStats `json:"endpoint_stats"`
}

type EndpointStats struct {
    Requests      int64         `json:"requests"`
    Errors        int64         `json:"errors"`
    AvgLatency    time.Duration `json:"avg_latency"`
    MinLatency    time.Duration `json:"min_latency"`
    MaxLatency    time.Duration `json:"max_latency"`
    LastRequest   time.Time     `json:"last_request"`
}
```

## HTTP Server Implementation

### Default HTTPServer
```go
type DefaultHTTPServer struct {
    config     *ServerConfig
    router     *mux.Router
    server     *http.Server
    handlers   map[string]Handler
    middleware []Middleware
    stats      *ServerStats
    mu         sync.RWMutex
    ctx        context.Context
    cancel     context.CancelFunc
    logger     *zap.Logger
    metrics    *ServerMetrics
}

func NewDefaultHTTPServer(config *ServerConfig, logger *zap.Logger) *DefaultHTTPServer {
    ctx, cancel := context.WithCancel(context.Background())
    
    router := mux.NewRouter()
    router.Use(mux.CORSMethodMiddleware(router))
    
    server := &http.Server{
        Addr:         fmt.Sprintf("%s:%d", config.ListenAddr, config.ListenPort),
        Handler:      router,
        ReadTimeout:  config.ReadTimeout,
        WriteTimeout: config.WriteTimeout,
        IdleTimeout:  config.IdleTimeout,
        MaxHeaderBytes: config.MaxHeaderBytes,
    }
    
    httpServer := &DefaultHTTPServer{
        config:   config,
        router:   router,
        server:   server,
        handlers: make(map[string]Handler),
        middleware: make([]Middleware, 0),
        stats: &ServerStats{
            EndpointStats: make(map[string]*EndpointStats),
        },
        ctx:     ctx,
        cancel:  cancel,
        logger:  logger,
        metrics: NewServerMetrics(),
    }
    
    // Регистрация стандартных middleware
    httpServer.registerDefaultMiddleware()
    
    // Регистрация стандартных обработчиков
    httpServer.registerDefaultHandlers()
    
    return httpServer
}

func (s *DefaultHTTPServer) Start(ctx context.Context) error {
    s.logger.Info("starting HTTP server",
        "addr", s.server.Addr,
        "tls_enabled", s.config.TLSEnabled)
    
    // Запуск сервера
    go func() {
        var err error
        
        if s.config.TLSEnabled {
            err = s.server.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
        } else {
            err = s.server.ListenAndServe()
        }
        
        if err != nil && err != http.ErrServerClosed {
            s.logger.Error("HTTP server error", "error", err)
        }
    }()
    
    // Запуск фоновых задач
    go s.statsCollectionLoop(ctx)
    go s.healthCheckLoop(ctx)
    
    s.logger.Info("HTTP server started successfully")
    return nil
}

func (s *DefaultHTTPServer) Stop(ctx context.Context) error {
    s.logger.Info("stopping HTTP server")
    
    // Отмена контекста
    s.cancel()
    
    // Graceful shutdown
    shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    err := s.server.Shutdown(shutdownCtx)
    if err != nil {
        s.logger.Error("HTTP server shutdown error", "error", err)
        return err
    }
    
    s.logger.Info("HTTP server stopped successfully")
    return nil
}

func (s *DefaultHTTPServer) RegisterHandler(ctx context.Context, pattern string, handler Handler) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Создание HTTP обработчика
    httpHandler := s.wrapHandler(handler)
    
    // Регистрация в роутере
    s.router.HandleFunc(pattern, httpHandler).Methods(handler.Method())
    
    // Сохранение обработчика
    s.handlers[pattern] = handler
    
    s.logger.Info("handler registered",
        "pattern", pattern,
        "method", handler.Method())
    
    return nil
}

func (s *DefaultHTTPServer) wrapHandler(handler Handler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // Создание контекста с request ID
        requestID := generateRequestID()
        ctx = context.WithValue(ctx, "request_id", requestID)
        
        // Логирование запроса
        s.logger.Debug("HTTP request",
            "method", r.Method,
            "url", r.URL.String(),
            "remote_addr", r.RemoteAddr,
            "user_agent", r.UserAgent(),
            "request_id", requestID)
        
        // Обновление статистики
        s.updateRequestStats(r.URL.Path, true)
        
        // Выполнение обработчика
        err := handler.Handle(ctx, w, r)
        if err != nil {
            s.handleError(w, r, err)
            s.updateRequestStats(r.URL.Path, false)
        }
        
        // Обновление метрик
        s.metrics.RecordRequest(r.URL.Path, r.Method, err == nil)
    }
}

func (s *DefaultHTTPServer) registerDefaultMiddleware() {
    // Recovery middleware
    s.RegisterMiddleware(context.Background(), NewRecoveryMiddleware(s.logger))
    
    // Logging middleware
    s.RegisterMiddleware(context.Background(), NewLoggingMiddleware(s.logger))
    
    // Metrics middleware
    s.RegisterMiddleware(context.Background(), NewMetricsMiddleware(s.metrics))
    
    // CORS middleware
    if s.config.CORSEnabled {
        s.RegisterMiddleware(context.Background(), NewCORSMiddleware(s.config))
    }
    
    // Rate limit middleware
    if s.config.RateLimitEnabled {
        s.RegisterMiddleware(context.Background(), NewRateLimitMiddleware(s.config))
    }
    
    // Auth middleware
    if s.config.AuthEnabled {
        s.RegisterMiddleware(context.Background(), NewAuthMiddleware(s.config))
    }
    
    // Timeout middleware
    s.RegisterMiddleware(context.Background(), NewTimeoutMiddleware(s.config.ReadTimeout))
}

func (s *DefaultHTTPServer) registerDefaultHandlers() {
    // Metrics handler
    metricsHandler := NewMetricsHandler(s.metrics)
    s.RegisterHandler(context.Background(), "/metrics", metricsHandler)
    
    // Health handler
    healthHandler := NewHealthHandler()
    s.RegisterHandler(context.Background(), "/health", healthHandler)
    
    // Readiness handler
    readinessHandler := NewReadinessHandler()
    s.RegisterHandler(context.Background(), "/ready", readinessHandler)
    
    // Debug handler
    debugHandler := NewDebugHandler()
    s.RegisterHandler(context.Background(), "/debug", debugHandler)
}
```

## Request Handlers

### Metrics Handler
```go
type MetricsHandler struct {
    metrics    *ServerMetrics
    exporter   *MetricsExporter
    mu         sync.RWMutex
}

func NewMetricsHandler(exporter *MetricsExporter) *MetricsHandler {
    return &MetricsHandler{
        exporter: exporter,
        metrics:  NewServerMetrics(),
    }
}

func (h *MetricsHandler) Handle(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    // Установка заголовков для Prometheus
    w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    
    // Получение метрик
    metricsData, err := h.exporter.Export(ctx)
    if err != nil {
        h.logger.Error("failed to export metrics", "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return err
    }
    
    // Запись метрик
    _, err = w.Write(metricsData)
    if err != nil {
        h.logger.Error("failed to write metrics", "error", err)
        return err
    }
    
    return nil
}

func (h *MetricsHandler) Method() string {
    return "GET"
}

func (h *MetricsHandler) Pattern() string {
    return "/metrics"
}

func (h *MetricsHandler) Middleware() []Middleware {
    return []Middleware{
        NewAuthMiddleware(nil), // Если требуется аутентификация
    }
}
```

### Health Handler
```go
type HealthHandler struct {
    checks     map[string]HealthCheck
    mu         sync.RWMutex
    logger     *zap.Logger
}

type HealthCheck interface {
    Check(ctx context.Context) error
    Name() string
}

func NewHealthHandler() *HealthHandler {
    return &HealthHandler{
        checks: make(map[string]HealthCheck),
        logger: zap.L(),
    }
}

func (h *HealthHandler) Handle(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    h.mu.RLock()
    checks := make(map[string]HealthCheck, len(h.checks))
    for k, v := range h.checks {
        checks[k] = v
    }
    h.mu.RUnlock()
    
    var failedChecks []string
    var checkErrors []string
    
    // Выполнение всех health checks
    for name, check := range checks {
        err := check.Check(ctx)
        if err != nil {
            failedChecks = append(failedChecks, name)
            checkErrors = append(checkErrors, fmt.Sprintf("%s: %v", name, err))
        }
    }
    
    // Определение статуса
    statusCode := http.StatusOK
    status := "healthy"
    
    if len(failedChecks) > 0 {
        statusCode = http.StatusServiceUnavailable
        status = "unhealthy"
    }
    
    // Создание ответа
    response := HealthResponse{
        Status:     status,
        Timestamp:  time.Now(),
        Checks:     make(map[string]string),
        Uptime:     time.Since(startTime),
        Version:    version,
        CommitHash: commitHash,
    }
    
    for name := range checks {
        if slices.Contains(failedChecks, name) {
            response.Checks[name] = "failed"
        } else {
            response.Checks[name] = "healthy"
        }
    }
    
    // Установка статуса и заголовков
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    
    // Запись JSON ответа
    encoder := json.NewEncoder(w)
    if err := encoder.Encode(response); err != nil {
        h.logger.Error("failed to encode health response", "error", err)
        return err
    }
    
    return nil
}

type HealthResponse struct {
    Status     string            `json:"status"`
    Timestamp  time.Time         `json:"timestamp"`
    Checks     map[string]string `json:"checks"`
    Uptime     time.Duration     `json:"uptime"`
    Version    string            `json:"version"`
    CommitHash string            `json:"commit_hash"`
}

func (h *HealthHandler) Method() string {
    return "GET"
}

func (h *HealthHandler) Pattern() string {
    return "/health"
}

func (h *HealthHandler) Middleware() []Middleware {
    return nil
}

func (h *HealthHandler) RegisterCheck(ctx context.Context, check HealthCheck) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    h.checks[check.Name()] = check
    return nil
}
```

### Readiness Handler
```go
type ReadinessHandler struct {
    readinessChecks map[string]ReadinessCheck
    mu              sync.RWMutex
    logger          *zap.Logger
}

type ReadinessCheck interface {
    IsReady(ctx context.Context) bool
    Name() string
    Description() string
}

func NewReadinessHandler() *ReadinessHandler {
    return &ReadinessHandler{
        readinessChecks: make(map[string]ReadinessCheck),
        logger:          zap.L(),
    }
}

func (r *ReadinessHandler) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
    r.mu.RLock()
    checks := make(map[string]ReadinessCheck, len(r.readinessChecks))
    for k, v := range r.readinessChecks {
        checks[k] = v
    }
    r.mu.RUnlock()
    
    var notReadyChecks []string
    var checkResults []CheckResult
    
    // Выполнение всех readiness checks
    for name, check := range checks {
        isReady := check.IsReady(ctx)
        
        result := CheckResult{
            Name:        name,
            Description: check.Description(),
            Ready:       isReady,
            Timestamp:   time.Now(),
        }
        
        checkResults = append(checkResults, result)
        
        if !isReady {
            notReadyChecks = append(notReadyChecks, name)
        }
    }
    
    // Определение статуса
    statusCode := http.StatusOK
    status := "ready"
    
    if len(notReadyChecks) > 0 {
        statusCode = http.StatusServiceUnavailable
        status = "not_ready"
    }
    
    // Создание ответа
    response := ReadinessResponse{
        Status:     status,
        Timestamp:  time.Now(),
        Checks:     checkResults,
        Uptime:     time.Since(startTime),
        Version:    version,
        CommitHash: commitHash,
    }
    
    // Установка заголовков
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    
    // Запись JSON ответа
    encoder := json.NewEncoder(w)
    if err := encoder.Encode(response); err != nil {
        r.logger.Error("failed to encode readiness response", "error", err)
        return err
    }
    
    return nil
}

type ReadinessResponse struct {
    Status     string        `json:"status"`
    Timestamp  time.Time     `json:"timestamp"`
    Checks     []CheckResult `json:"checks"`
    Uptime     time.Duration `json:"uptime"`
    Version    string        `json:"version"`
    CommitHash string        `json:"commit_hash"`
}

type CheckResult struct {
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Ready       bool      `json:"ready"`
    Timestamp   time.Time `json:"timestamp"`
}

func (r *ReadinessHandler) Method() string {
    return "GET"
}

func (r *ReadinessHandler) Pattern() string {
    return "/ready"
}

func (r *ReadinessHandler) Middleware() []Middleware {
    return nil
}

func (r *ReadinessHandler) RegisterCheck(ctx context.Context, check ReadinessCheck) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    r.readinessChecks[check.Name()] = check
    return nil
}
```

## Middleware Implementation

### Logging Middleware
```go
type LoggingMiddleware struct {
    logger *zap.Logger
    mu     sync.RWMutex
}

func NewLoggingMiddleware(logger *zap.Logger) *LoggingMiddleware {
    return &LoggingMiddleware{
        logger: logger,
    }
}

func (m *LoggingMiddleware) Handle(ctx context.Context, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Создание response writer wrapper для логирования ответа
        responseWriter := &responseWriter{
            ResponseWriter: w,
            statusCode:     http.StatusOK,
            bytesWritten:   0,
        }
        
        // Логирование входящего запроса
        m.logger.Info("HTTP request started",
            "method", r.Method,
            "url", r.URL.String(),
            "remote_addr", r.RemoteAddr,
            "user_agent", r.UserAgent(),
            "request_id", getRequestID(ctx))
        
        // Выполнение следующего обработчика
        next.ServeHTTP(responseWriter, r)
        
        // Логирование завершения запроса
        duration := time.Since(start)
        
        m.logger.Info("HTTP request completed",
            "method", r.Method,
            "url", r.URL.String(),
            "status_code", responseWriter.statusCode,
            "bytes_written", responseWriter.bytesWritten,
            "duration", duration,
            "request_id", getRequestID(ctx))
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode   int
    bytesWritten int64
}

func (rw *responseWriter) WriteHeader(statusCode int) {
    rw.statusCode = statusCode
    rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
    n, err := rw.ResponseWriter.Write(b)
    rw.bytesWritten += int64(n)
    return n, err
}

func (m *LoggingMiddleware) Name() string {
    return "logging"
}

func (m *LoggingMiddleware) Priority() int {
    return 100
}
```

### Metrics Middleware
```go
type MetricsMiddleware struct {
    metrics *ServerMetrics
    mu      sync.RWMutex
}

func NewMetricsMiddleware(metrics *ServerMetrics) *MetricsMiddleware {
    return &MetricsMiddleware{
        metrics: metrics,
    }
}

func (m *MetricsMiddleware) Handle(ctx context.Context, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Создание response writer wrapper для сбора метрик
        responseWriter := &metricsResponseWriter{
            ResponseWriter: w,
            statusCode:     http.StatusOK,
        }
        
        // Выполнение следующего обработчика
        next.ServeHTTP(responseWriter, r)
        
        // Обновление метрик
        duration := time.Since(start)
        m.metrics.RecordRequest(r.URL.Path, r.Method, responseWriter.statusCode < 400)
        m.metrics.RecordResponseTime(r.URL.Path, duration)
    })
}

type metricsResponseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *metricsResponseWriter) WriteHeader(statusCode int) {
    rw.statusCode = statusCode
    rw.ResponseWriter.WriteHeader(statusCode)
}

func (m *MetricsMiddleware) Name() string {
    return "metrics"
}

func (m *MetricsMiddleware) Priority() int {
    return 200
}
```

### Rate Limit Middleware
```go
type RateLimitMiddleware struct {
    config  *ServerConfig
    limiter *rate.Limiter
    mu      sync.RWMutex
}

func NewRateLimitMiddleware(config *ServerConfig) *RateLimitMiddleware {
    limiter := rate.NewLimiter(rate.Limit(config.RateLimitRPS), config.RateLimitBurst)
    
    return &RateLimitMiddleware{
        config:  config,
        limiter: limiter,
    }
}

func (m *RateLimitMiddleware) Handle(ctx context.Context, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Проверка rate limit
        if !m.limiter.Allow() {
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
        
        // Выполнение следующего обработчика
        next.ServeHTTP(w, r)
    })
}

func (m *RateLimitMiddleware) Name() string {
    return "rate_limit"
}

func (m *RateLimitMiddleware) Priority() int {
    return 300
}
```

### CORS Middleware
```go
type CORSMiddleware struct {
    config *ServerConfig
}

func NewCORSMiddleware(config *ServerConfig) *CORSMiddleware {
    return &CORSMiddleware{
        config: config,
    }
}

func (m *CORSMiddleware) Handle(ctx context.Context, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Установка CORS заголовков
        origin := r.Header.Get("Origin")
        
        if m.isAllowedOrigin(origin) {
            w.Header().Set("Access-Control-Allow-Origin", origin)
        } else if len(m.config.CORSOrigins) == 0 {
            w.Header().Set("Access-Control-Allow-Origin", "*")
        }
        
        w.Header().Set("Access-Control-Allow-Methods", strings.Join(m.config.CORSMethods, ", "))
        w.Header().Set("Access-Control-Allow-Headers", strings.Join(m.config.CORSHeaders, ", "))
        w.Header().Set("Access-Control-Max-Age", "86400")
        
        // Обработка preflight запросов
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        // Выполнение следующего обработчика
        next.ServeHTTP(w, r)
    })
}

func (m *CORSMiddleware) isAllowedOrigin(origin string) bool {
    if len(m.config.CORSOrigins) == 0 {
        return true
    }
    
    for _, allowedOrigin := range m.config.CORSOrigins {
        if origin == allowedOrigin {
            return true
        }
    }
    
    return false
}

func (m *CORSMiddleware) Name() string {
    return "cors"
}

func (m *CORSMiddleware) Priority() int {
    return 50
}
```

### Recovery Middleware
```go
type RecoveryMiddleware struct {
    logger *zap.Logger
}

func NewRecoveryMiddleware(logger *zap.Logger) *RecoveryMiddleware {
    return &RecoveryMiddleware{
        logger: logger,
    }
}

func (m *RecoveryMiddleware) Handle(ctx context.Context, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if r := recover(); r != nil {
                // Логирование паники
                m.logger.Error("HTTP handler panic",
                    "panic", r,
                    "stack", string(debug.Stack()),
                    "url", r.URL.String(),
                    "method", r.Method)
                
                // Отправка 500 ошибки
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        
        // Выполнение следующего обработчика
        next.ServeHTTP(w, r)
    })
}

func (m *RecoveryMiddleware) Name() string {
    return "recovery"
}

func (m *RecoveryMiddleware) Priority() int {
    return 10
}
```

## Server Metrics

### ServerMetrics
```go
type ServerMetrics struct {
    requestsTotal     prometheus.CounterVec
    requestDuration   prometheus.HistogramVec
    activeConnections prometheus.Gauge
    memoryUsage       prometheus.Gauge
    cpuUsage          prometheus.Gauge
    mu                sync.RWMutex
}

func NewServerMetrics() *ServerMetrics {
    return &ServerMetrics{
        requestsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "vmprober_http_requests_total",
                Help: "Total number of HTTP requests",
            },
            []string{"endpoint", "method", "status"},
        ),
        requestDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "vmprober_http_request_duration_seconds",
                Help:    "Duration of HTTP requests",
                Buckets: prometheus.DefBuckets,
            },
            []string{"endpoint", "method"},
        ),
        activeConnections: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "vmprober_http_active_connections",
                Help: "Number of active HTTP connections",
            },
        ),
        memoryUsage: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "vmprober_http_memory_usage_bytes",
                Help: "Memory usage of HTTP server",
            },
        ),
        cpuUsage: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "vmprober_http_cpu_usage_percent",
                Help: "CPU usage of HTTP server",
            },
        ),
    }
}

func (m *ServerMetrics) RecordRequest(endpoint, method string, success bool) {
    status := "success"
    if !success {
        status = "error"
    }
    
    m.requestsTotal.WithLabelValues(endpoint, method, status).Inc()
}

func (m *ServerMetrics) RecordResponseTime(endpoint, method string, duration time.Duration) {
    m.requestDuration.WithLabelValues(endpoint, method).Observe(duration.Seconds())
}

func (m *ServerMetrics) UpdateConnectionCount(count int64) {
    m.activeConnections.Set(float64(count))
}

func (m *ServerMetrics) UpdateMemoryUsage(bytes uint64) {
    m.memoryUsage.Set(float64(bytes))
}

func (m *ServerMetrics) UpdateCPUUsage(percent float64) {
    m.cpuUsage.Set(percent)
}
```

## Configuration Examples

### Basic Configuration
```yaml
listen:
  addr: "0.0.0.0"
  port: 8429
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
  max_header_bytes: 1048576
```

### Advanced Configuration
```yaml
listen:
  addr: "0.0.0.0"
  port: 8429
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
  max_header_bytes: 1048576
  
  # TLS Configuration
  tls_enabled: true
  tls_cert_file: "/path/to/cert.pem"
  tls_key_file: "/path/to/key.pem"
  tls_min_version: "1.2"
  
  # CORS Configuration
  cors_enabled: true
  cors_origins:
    - "https://grafana.example.com"
    - "https://monitoring.example.com"
  cors_methods:
    - "GET"
    - "POST"
    - "OPTIONS"
  cors_headers:
    - "Content-Type"
    - "Authorization"
    - "X-Requested-With"
  
  # Rate Limiting
  rate_limit_enabled: true
  rate_limit_rps: 100.0
  rate_limit_burst: 200
  
  # Security
  auth_enabled: true
  auth_type: "bearer"
  auth_config:
    token_file: "/path/to/tokens.txt"
  
  # Health Checks
  health_check_interval: 30s
  readiness_check_interval: 10s
```

## Security Considerations

### 1. TLS Configuration
- Использование современных версий TLS (1.2+)
- Автоматическое обновление сертификатов
- HSTS заголовки

### 2. Authentication
- Bearer token authentication
- API key authentication
- OAuth2 support (опционально)

### 3. Rate Limiting
- Per-IP rate limiting
- Global rate limiting
- Burst protection

### 4. Security Headers
- Content Security Policy
- X-Frame-Options
- X-Content-Type-Options
- Strict-Transport-Security

### 5. Request Validation
- Input sanitization
- Size limits
- Method validation

## Performance Optimizations

### 1. Connection Management
- Keep-alive connections
- Connection pooling
- Timeout management

### 2. Request Processing
- Async request handling
- Batch processing
- Caching strategies

### 3. Resource Management
- Memory pooling
- Garbage collection optimization
- CPU usage monitoring

### 4. Monitoring
- Real-time metrics
- Performance profiling
- Alerting integration

## Testing Strategy

### 1. Unit Tests
- Handler testing
- Middleware testing
- Configuration validation

### 2. Integration Tests
- End-to-end testing
- Load testing
- Security testing

### 3. Performance Tests
- Benchmark testing
- Stress testing
- Scalability testing