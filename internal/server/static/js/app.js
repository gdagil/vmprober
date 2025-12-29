// VMProber Dashboard JavaScript

let autoRefreshInterval;
let currentFilter = 'all';
let allJobs = [];

function formatDuration(duration) {
    if (typeof duration === 'number') {
        const seconds = Math.floor(duration / 1000000000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);
        const secs = seconds % 60;
        const mins = minutes % 60;
        const hrs = hours % 24;
        
        const parts = [];
        if (days > 0) parts.push(days + 'd');
        if (hrs > 0) parts.push(hrs + 'h');
        if (mins > 0) parts.push(mins + 'm');
        if (secs > 0 || parts.length === 0) parts.push(secs + 's');
        return parts.join(' ');
    }
    
    if (typeof duration === 'string') {
        const cleanDuration = duration.replace(/\.\d+s/, 's');
        const parts = [];
        
        const days = cleanDuration.match(/(\d+)d/);
        const hours = cleanDuration.match(/(\d+)h/);
        const minutes = cleanDuration.match(/(\d+)m/);
        const seconds = cleanDuration.match(/(\d+)s/);
        
        if (days) parts.push(days[1] + 'd');
        if (hours) parts.push(hours[1] + 'h');
        if (minutes) parts.push(minutes[1] + 'm');
        if (seconds) parts.push(seconds[1] + 's');
        
        return parts.join(' ') || '0s';
    }
    
    return '0s';
}

function formatTime(timeStr) {
    if (!timeStr || timeStr === '0001-01-01T00:00:00Z') return '—';
    const date = new Date(timeStr);
    const now = new Date();
    const diff = now - date;
    
    if (diff < 60000) return 'just now';
    if (diff < 3600000) return Math.floor(diff / 60000) + 'm ago';
    if (diff < 86400000) return Math.floor(diff / 3600000) + 'h ago';
    
    return date.toLocaleString('ru-RU', { 
        day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' 
    });
}

function formatTimeDetailed(timeStr) {
    if (!timeStr || timeStr === '0001-01-01T00:00:00Z') return '—';
    const date = new Date(timeStr);
    return date.toLocaleString('ru-RU', {
        day: '2-digit', month: 'short', year: 'numeric',
        hour: '2-digit', minute: '2-digit', second: '2-digit'
    });
}

async function loadStats() {
    try {
        const response = await fetch('/api/v1/stats');
        const data = await response.json();
        
        const total = data.scheduler.total_jobs || 0;
        const running = data.scheduler.running_jobs || 0;
        const failed = data.scheduler.failed_jobs || 0;
        const up = total - failed;
        
        document.getElementById('totalJobs').textContent = total;
        document.getElementById('runningJobs').textContent = running;
        document.getElementById('failedJobs').textContent = failed;
        document.getElementById('upCount').textContent = up > 0 ? up : 0;
        document.getElementById('uptime').textContent = formatDuration(data.uptime || '0s');
        
        // Success rate
        const successRate = total > 0 ? Math.round((up / total) * 100) : 0;
        document.getElementById('successRate').textContent = successRate + '%';
        
        // Health bar
        const upPct = total > 0 ? (up / total) * 100 : 0;
        const downPct = total > 0 ? (failed / total) * 100 : 0;
        const unknownPct = 100 - upPct - downPct;
        
        document.getElementById('healthUp').style.width = upPct + '%';
        document.getElementById('healthDown').style.width = downPct + '%';
        document.getElementById('healthUnknown').style.width = unknownPct + '%';
        document.getElementById('healthUpPct').textContent = Math.round(upPct) + '%';
        document.getElementById('healthDownPct').textContent = Math.round(downPct) + '%';
        document.getElementById('healthUnknownPct').textContent = Math.round(unknownPct) + '%';
        
        // Last update
        document.getElementById('lastUpdate').textContent = 'Last update: ' + new Date().toLocaleTimeString('ru-RU');
    } catch (error) {
        console.error('Error loading stats:', error);
    }
}

async function loadJobs() {
    try {
        const response = await fetch('/api/v1/jobs');
        allJobs = await response.json() || [];
        
        document.getElementById('jobsCount').textContent = allJobs.length;
        
        // Protocol counts
        const tcpCount = allJobs.filter(j => j.target?.protocol === 'tcp').length;
        const udpCount = allJobs.filter(j => j.target?.protocol === 'udp').length;
        const icmpCount = allJobs.filter(j => j.target?.protocol === 'icmp').length;
        
        document.querySelector('#protocolTCP .vm-protocol-card__count').textContent = tcpCount;
        document.querySelector('#protocolUDP .vm-protocol-card__count').textContent = udpCount;
        document.querySelector('#protocolICMP .vm-protocol-card__count').textContent = icmpCount;
        
        renderJobs();
    } catch (error) {
        document.getElementById('jobsContent').innerHTML = '<div class="vm-error"><svg viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg> Error: ' + error.message + '</div>';
    }
}

function renderJobs() {
    const searchPhrase = document.getElementById('search').value.toLowerCase();
    let filteredJobs = allJobs;
    
    // Apply status filter
    if (currentFilter === 'up') {
        filteredJobs = filteredJobs.filter(j => j.last_status === 'up');
    } else if (currentFilter === 'down') {
        filteredJobs = filteredJobs.filter(j => j.last_status === 'down');
    }
    
    // Apply search filter
    if (searchPhrase) {
        filteredJobs = filteredJobs.filter(j => 
            j.id?.toLowerCase().includes(searchPhrase) ||
            j.target?.host?.toLowerCase().includes(searchPhrase) ||
            j.target?.protocol?.toLowerCase().includes(searchPhrase)
        );
    }
    
    if (filteredJobs.length === 0) {
        document.getElementById('jobsContent').innerHTML = '<div class="vm-empty"><svg viewBox="0 0 24 24"><path d="M19 5v14H5V5h14m0-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/></svg><span>No probes match your criteria</span></div>';
        return;
    }

    let html = '<div class="vm-jobs-grid">';

    filteredJobs.forEach(job => {
        const status = job.last_status || 'unknown';
        const statusClass = status === 'up' ? 'up' : (status === 'down' ? 'down' : 'unknown');
        const statusIcon = status === 'up' ? '✓' : (status === 'down' ? '✗' : '?');
        const successCount = job.success_count || 0;
        const failedCount = job.failed_count || 0;
        const totalProbes = successCount + failedCount;
        const successPct = totalProbes > 0 ? Math.round((successCount / totalProbes) * 100) : 0;
        
        html += '<div class="vm-job-card vm-job-card--' + statusClass + '">';
        html += '<div class="vm-job-card__header">';
        html += '<div class="vm-job-card__status vm-job-card__status--' + statusClass + '">' + statusIcon + '</div>';
        html += '<div class="vm-job-card__info">';
        html += '<div class="vm-job-card__target">' + (job.target?.host || 'unknown') + ':' + (job.target?.port || 'N/A') + '</div>';
        html += '<div class="vm-job-card__protocol"><span class="vm-protocol-tag vm-protocol-tag--' + (job.target?.protocol || 'unknown') + '">' + (job.target?.protocol || 'N/A').toUpperCase() + '</span></div>';
        html += '</div>';
        html += '</div>';
        html += '<div class="vm-job-card__body">';
        html += '<div class="vm-job-card__stats">';
        html += '<div class="vm-job-card__stat"><span class="vm-job-card__stat-value vm-job-card__stat-value--success">' + successCount + '</span><span class="vm-job-card__stat-label">Success</span></div>';
        html += '<div class="vm-job-card__stat"><span class="vm-job-card__stat-value vm-job-card__stat-value--failed">' + failedCount + '</span><span class="vm-job-card__stat-label">Failed</span></div>';
        html += '<div class="vm-job-card__stat"><span class="vm-job-card__stat-value">' + successPct + '%</span><span class="vm-job-card__stat-label">Rate</span></div>';
        html += '</div>';
        html += '<div class="vm-job-card__progress"><div class="vm-job-card__progress-bar" style="width: ' + successPct + '%"></div></div>';
        html += '<div class="vm-job-card__meta">';
        html += '<span class="vm-job-card__interval">⏱ ' + formatDuration(job.interval || '0s') + '</span>';
        html += '<span class="vm-job-card__time">🕒 ' + formatTime(job.last_probe_time) + '</span>';
        html += '</div>';
        html += '</div>';
        html += '</div>';
    });

    html += '</div>';
    document.getElementById('jobsContent').innerHTML = html;
}

async function loadTargets() {
    try {
        const response = await fetch('/api/v1/targets');
        const targets = await response.json() || [];
        
        document.getElementById('targetsCount').textContent = targets.length;
        
        if (targets.length === 0) {
            document.getElementById('targetsContent').innerHTML = '<div class="vm-empty"><svg viewBox="0 0 24 24"><path d="M19 5v14H5V5h14m0-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/></svg><span>No targets configured</span></div>';
            return;
        }

        let html = '<table class="vm-table"><thead><tr>';
        html += '<th>Status</th><th>Host</th><th>Port</th><th>Protocol</th><th>Interval</th><th>Timeout</th><th>Success</th><th>Failed</th><th>Last Probe</th><th>Labels</th>';
        html += '</tr></thead><tbody>';

        targets.forEach(target => {
            const status = target.status || 'unknown';
            const statusClass = status === 'up' ? 'up' : (status === 'down' ? 'down' : 'unknown');
            const statusIcon = status === 'up' ? '✓' : (status === 'down' ? '✗' : '?');
            
            html += '<tr class="vm-table__row--' + statusClass + '">';
            html += '<td><span class="vm-status-badge vm-status-badge--' + statusClass + '">' + statusIcon + ' ' + status.toUpperCase() + '</span></td>';
            html += '<td class="vm-table__cell--host">' + target.host + '</td>';
            html += '<td>' + (target.port || '—') + '</td>';
            html += '<td><span class="vm-protocol-tag vm-protocol-tag--' + target.protocol + '">' + target.protocol.toUpperCase() + '</span></td>';
            html += '<td>' + formatDuration(target.interval) + '</td>';
            html += '<td>' + formatDuration(target.timeout) + '</td>';
            html += '<td class="vm-table__cell--success">' + (target.success_count || 0) + '</td>';
            html += '<td class="vm-table__cell--failed">' + (target.failed_count || 0) + '</td>';
            html += '<td class="vm-table__cell--time">' + formatTimeDetailed(target.last_probe_time) + '</td>';
            html += '<td>';
            if (target.labels && Object.keys(target.labels).length > 0) {
                html += '<div class="vm-labels">';
                for (const [key, value] of Object.entries(target.labels)) {
                    html += '<span class="vm-label">' + key + '="' + value + '"</span>';
                }
                html += '</div>';
            } else {
                html += '<span class="vm-table__cell--empty">—</span>';
            }
            html += '</td>';
            html += '</tr>';
        });

        html += '</tbody></table>';
        document.getElementById('targetsContent').innerHTML = html;
    } catch (error) {
        document.getElementById('targetsContent').innerHTML = '<div class="vm-error"><svg viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg> Error: ' + error.message + '</div>';
    }
}

async function loadData() {
    const btn = document.getElementById('refreshBtn');
    if (btn) btn.classList.add('vm-button--loading');
    
    await Promise.all([loadStats(), loadJobs(), loadTargets()]);
    
    if (btn) btn.classList.remove('vm-button--loading');
}

function startAutoRefresh() {
    autoRefreshInterval = setInterval(loadData, 5000);
}

function stopAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }
}

// Event Listeners
window.addEventListener('load', () => {
    loadData();
    startAutoRefresh();
    
    // Search
    document.getElementById('search').addEventListener('input', renderJobs);
    
    // Filter buttons
    document.querySelectorAll('.vm-filter-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.vm-filter-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            currentFilter = btn.dataset.filter;
            renderJobs();
        });
    });
    
    // Auto-refresh toggle
    document.getElementById('autoRefreshToggle').addEventListener('change', (e) => {
        if (e.target.checked) {
            startAutoRefresh();
        } else {
            stopAutoRefresh();
        }
    });
});

window.addEventListener('beforeunload', stopAutoRefresh);

