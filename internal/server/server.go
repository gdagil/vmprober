package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/vmprober/vmprober/internal/metrics"
	"github.com/vmprober/vmprober/internal/scheduler"
)

//go:embed static
var staticFiles embed.FS

// Server HTTP сервер для метрик и health checks
type Server struct {
	httpServer      *http.Server
	logger          *logrus.Logger
	startTime       time.Time
	scheduler       *scheduler.Scheduler
	metricsCollector *metrics.Collector
}

// NewServer создает новый HTTP сервер
func NewServer(host string, port int, logger *logrus.Logger) *Server {
	mux := http.NewServeMux()
	server := &Server{
		logger:    logger,
		startTime: time.Now(),
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", host, port),
			Handler:      mux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
	}

	// Создаем sub-filesystem для статических файлов
	staticFS, _ := fs.Sub(staticFiles, "static")

	// Регистрация handlers
	mux.HandleFunc("/health", server.healthHandler)
	mux.HandleFunc("/ready", server.readyHandler)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/api/v1/jobs", server.jobsHandler)
	mux.HandleFunc("/api/v1/stats", server.statsHandler)
	mux.HandleFunc("/api/v1/targets", server.targetsHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/", server.uiHandler)

	return server
}

// SetScheduler устанавливает планировщик
func (s *Server) SetScheduler(sched *scheduler.Scheduler) {
	s.scheduler = sched
}

// SetMetricsCollector устанавливает коллектор метрик
func (s *Server) SetMetricsCollector(collector *metrics.Collector) {
	s.metricsCollector = collector
}

// Start запускает HTTP сервер
func (s *Server) Start(ctx context.Context) error {
	s.logger.WithFields(logrus.Fields{
		"addr": s.httpServer.Addr,
	}).Info("Starting HTTP server")

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Error("HTTP server error")
		}
	}()

	return nil
}

// Stop останавливает HTTP сервер
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")
	return s.httpServer.Shutdown(ctx)
}

// healthHandler обрабатывает /health endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","uptime":"%s"}`, time.Since(s.startTime))
}

// readyHandler обрабатывает /ready endpoint
func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"ready":true}`)
}

// jobsHandler обрабатывает /api/v1/jobs endpoint
func (s *Server) jobsHandler(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		http.Error(w, "Scheduler not initialized", http.StatusInternalServerError)
		return
	}

	jobs := s.scheduler.GetAllJobs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

// statsHandler обрабатывает /api/v1/stats endpoint
func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		http.Error(w, "Scheduler not initialized", http.StatusInternalServerError)
		return
	}

	stats := s.scheduler.GetStats()
	response := map[string]interface{}{
		"scheduler": stats,
		"uptime":    time.Since(s.startTime).String(),
		"start_time": s.startTime.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// targetsHandler обрабатывает /api/v1/targets endpoint
func (s *Server) targetsHandler(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		http.Error(w, "Scheduler not initialized", http.StatusInternalServerError)
		return
	}

	jobs := s.scheduler.GetAllJobs()
	targets := make([]map[string]interface{}, 0, len(jobs))
	
	for _, job := range jobs {
		targets = append(targets, map[string]interface{}{
			"id":       job.ID,
			"host":     job.Target.Host,
			"port":     job.Target.Port,
			"protocol": job.Target.Protocol,
			"interval": job.Interval.String(),
			"timeout":  job.Target.Timeout.String(),
			"labels":   job.Target.Labels,
			"enabled":  job.Target.Enabled,
			"next_run": job.NextRun.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targets)
}

// uiHandler обрабатывает главную страницу UI
func (s *Server) uiHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, getUIHTML())
}

// getUIHTML возвращает HTML код UI в стиле VictoriaMetrics VMUI
func getUIHTML() string {
	return `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>vmprober</title>
    <link href="/static/css/custom.css" rel="stylesheet">
</head>
<body>
    <header class="vm-header">
        <a href="/" class="vm-header__logo">
            <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
            </svg>
            vmprober
        </a>
        <nav class="vm-header__nav">
            <a href="/" class="vm-header__link active">Status</a>
            <a href="/api/v1/targets" class="vm-header__link">Targets</a>
            <a href="/api/v1/jobs" class="vm-header__link">Jobs</a>
            <a href="/metrics" class="vm-header__link">Metrics</a>
            <a href="/api/v1/stats" class="vm-header__link">Stats</a>
        </nav>
    </header>

    <main class="vm-container">
        <div class="vm-stats-grid" id="statsGrid">
            <div class="vm-stat-card">
                <div class="vm-stat-card__value" id="totalJobs">-</div>
                <div class="vm-stat-card__label">Total Jobs</div>
            </div>
            <div class="vm-stat-card">
                <div class="vm-stat-card__value" id="runningJobs">-</div>
                <div class="vm-stat-card__label">Running</div>
            </div>
            <div class="vm-stat-card">
                <div class="vm-stat-card__value" id="queuedJobs">-</div>
                <div class="vm-stat-card__label">Queued</div>
            </div>
            <div class="vm-stat-card">
                <div class="vm-stat-card__value" id="completedJobs">-</div>
                <div class="vm-stat-card__label">Completed</div>
            </div>
            <div class="vm-stat-card">
                <div class="vm-stat-card__value" id="failedJobs">-</div>
                <div class="vm-stat-card__label">Failed</div>
            </div>
            <div class="vm-stat-card">
                <div class="vm-stat-card__value" id="uptime">-</div>
                <div class="vm-stat-card__label">Uptime</div>
            </div>
        </div>

        <div class="vm-card">
            <div class="vm-card__header">
                <h2 class="vm-card__title">Probe Jobs</h2>
                <div class="vm-toolbar">
                    <div class="vm-search">
                        <svg class="vm-search__icon" viewBox="0 0 24 24">
                            <path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
                        </svg>
                        <input type="text" class="vm-search__input" id="search" placeholder="Filter jobs...">
                    </div>
                    <button class="vm-button vm-button--outlined" onclick="loadData()" id="refreshBtn">
                        <svg viewBox="0 0 24 24">
                            <path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/>
                        </svg>
                        Refresh
                    </button>
                </div>
            </div>
            <div class="vm-card__body" id="jobsContent">
                <div class="vm-loading">Loading jobs...</div>
            </div>
        </div>

        <div class="vm-card">
            <div class="vm-card__header">
                <h2 class="vm-card__title">Targets</h2>
            </div>
            <div class="vm-card__body" id="targetsContent">
                <div class="vm-loading">Loading targets...</div>
            </div>
        </div>
    </main>

    <script>
        let autoRefreshInterval;

        function formatDuration(duration) {
            if (typeof duration === 'number') {
                const seconds = Math.floor(duration / 1000000000);
                const minutes = Math.floor(seconds / 60);
                const hours = Math.floor(minutes / 60);
                const secs = seconds % 60;
                const mins = minutes % 60;
                
                const parts = [];
                if (hours > 0) parts.push(hours + 'h');
                if (mins > 0) parts.push(mins + 'm');
                if (secs > 0 || parts.length === 0) parts.push(secs + 's');
                return parts.join(' ');
            }
            
            if (typeof duration === 'string') {
                const cleanDuration = duration.replace(/\.\d+s/, 's');
                const parts = [];
                
                const hours = cleanDuration.match(/(\d+)h/);
                const minutes = cleanDuration.match(/(\d+)m/);
                const seconds = cleanDuration.match(/(\d+)s/);
                
                if (hours) parts.push(hours[1] + 'h');
                if (minutes) parts.push(minutes[1] + 'm');
                if (seconds) parts.push(seconds[1] + 's');
                
                return parts.join(' ') || '0s';
            }
            
            return '0s';
        }

        function formatTime(timeStr) {
            const date = new Date(timeStr);
            return date.toLocaleString('ru-RU');
        }

        async function loadStats() {
            try {
                const response = await fetch('/api/v1/stats');
                const data = await response.json();
                
                document.getElementById('totalJobs').textContent = data.scheduler.total_jobs || 0;
                document.getElementById('runningJobs').textContent = data.scheduler.running_jobs || 0;
                document.getElementById('queuedJobs').textContent = data.scheduler.queued_jobs || 0;
                document.getElementById('completedJobs').textContent = data.scheduler.completed_jobs || 0;
                document.getElementById('failedJobs').textContent = data.scheduler.failed_jobs || 0;
                document.getElementById('uptime').textContent = formatDuration(data.uptime || '0s');
            } catch (error) {
                console.error('Error loading stats:', error);
            }
        }

        async function loadJobs() {
            try {
                const response = await fetch('/api/v1/jobs');
                const jobs = await response.json();
                
                if (!jobs || jobs.length === 0) {
                    document.getElementById('jobsContent').innerHTML = '<div class="vm-empty">No jobs configured</div>';
                    return;
                }

                let html = '<table class="vm-table"><thead><tr>';
                html += '<th>ID</th><th>Target</th><th>Protocol</th><th>Interval</th><th>Next Run</th><th>Priority</th>';
                html += '</tr></thead><tbody>';

                jobs.forEach(job => {
                    html += '<tr>';
                    html += '<td><code>' + job.id + '</code></td>';
                    html += '<td>' + job.target.host + ':' + (job.target.port || 'N/A') + '</td>';
                    html += '<td><span class="vm-badge vm-badge--info">' + (job.target.protocol || 'N/A').toUpperCase() + '</span></td>';
                    html += '<td>' + formatDuration(job.interval || '0s') + '</td>';
                    html += '<td class="vm-timestamp">' + formatTime(job.next_run) + '</td>';
                    html += '<td>' + (job.priority || 0) + '</td>';
                    html += '</tr>';
                });

                html += '</tbody></table>';
                document.getElementById('jobsContent').innerHTML = html;
            } catch (error) {
                document.getElementById('jobsContent').innerHTML = '<div class="vm-error">Error loading jobs: ' + error.message + '</div>';
            }
        }

        async function loadTargets() {
            try {
                const response = await fetch('/api/v1/targets');
                const targets = await response.json();
                
                if (!targets || targets.length === 0) {
                    document.getElementById('targetsContent').innerHTML = '<div class="vm-empty">No targets configured</div>';
                    return;
                }

                let html = '<table class="vm-table"><thead><tr>';
                html += '<th>ID</th><th>Host</th><th>Port</th><th>Protocol</th><th>Interval</th><th>Timeout</th><th>Status</th><th>Labels</th>';
                html += '</tr></thead><tbody>';

                targets.forEach(target => {
                    html += '<tr>';
                    html += '<td><code>' + target.id + '</code></td>';
                    html += '<td>' + target.host + '</td>';
                    html += '<td>' + (target.port || 'N/A') + '</td>';
                    html += '<td><span class="vm-badge vm-badge--info">' + target.protocol.toUpperCase() + '</span></td>';
                    html += '<td>' + formatDuration(target.interval) + '</td>';
                    html += '<td>' + formatDuration(target.timeout) + '</td>';
                    html += '<td>';
                    if (target.enabled) {
                        html += '<span class="vm-status-dot vm-status-dot--up"></span><span class="vm-badge vm-badge--success">UP</span>';
                    } else {
                        html += '<span class="vm-status-dot vm-status-dot--down"></span><span class="vm-badge vm-badge--error">DOWN</span>';
                    }
                    html += '</td>';
                    html += '<td>';
                    if (target.labels && Object.keys(target.labels).length > 0) {
                        html += '<div class="vm-labels">';
                        for (const [key, value] of Object.entries(target.labels)) {
                            html += '<span class="vm-label">' + key + '="' + value + '"</span>';
                        }
                        html += '</div>';
                    } else {
                        html += '<span style="color: #9E9E9E;">—</span>';
                    }
                    html += '</td>';
                    html += '</tr>';
                });

                html += '</tbody></table>';
                document.getElementById('targetsContent').innerHTML = html;
            } catch (error) {
                document.getElementById('targetsContent').innerHTML = '<div class="vm-error">Error loading targets: ' + error.message + '</div>';
            }
        }

        async function loadData() {
            const btn = document.getElementById('refreshBtn');
            if (btn) {
                btn.disabled = true;
                btn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/></svg> Loading...';
            }
            
            await Promise.all([loadStats(), loadJobs(), loadTargets()]);
            
            if (btn) {
                btn.disabled = false;
                btn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/></svg> Refresh';
            }
        }

        function filterJobs() {
            const searchPhrase = document.getElementById('search').value.toLowerCase();
            document.querySelectorAll('.vm-table tbody tr').forEach(row => {
                const text = row.innerText.toLowerCase();
                row.style.display = text.includes(searchPhrase) ? '' : 'none';
            });
        }

        function startAutoRefresh() {
            autoRefreshInterval = setInterval(loadData, 5000);
        }

        function stopAutoRefresh() {
            if (autoRefreshInterval) {
                clearInterval(autoRefreshInterval);
            }
        }

        window.addEventListener('load', () => {
            loadData();
            startAutoRefresh();
            
            const searchInput = document.getElementById('search');
            if (searchInput) {
                searchInput.addEventListener('input', filterJobs);
            }
        });

        window.addEventListener('beforeunload', stopAutoRefresh);
    </script>
</body>
</html>`
}

