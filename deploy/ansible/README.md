# Ansible Playbooks для vmprober

Этот набор Ansible playbook'ов предназначен для установки, настройки и управления vmprober на целевых хостах.

## Особенности

- Модульная структура с использованием ролей
- Автоматическая генерация проб для всех нод из inventory
- Разделение задач по отдельным файлам
- Гибкое управление этапами развертывания
- Поддержка TCP и ICMP проб для всех нод
- Автоматическое определение архитектуры системы
- Поддержка полной конфигурации vmprober (push, pull, scheduler)
- Гибкая настройка проб (TCP, UDP, ICMP, HTTP, DNS, gRPC)

## Структура файлов

```
deploy/ansible/
├── ansible.cfg                 # Конфигурация Ansible
├── inventory.yml               # Inventory файл
├── group_vars/
│   └── all.yml                 # Переменные для всех хостов
├── playbooks/
│   ├── deploy.yml              # Полное развертывание (установка + конфигурация + сервис)
│   ├── install.yml             # Только установка
│   ├── configure.yml           # Только конфигурация
│   └── service.yml             # Только управление сервисом
└── roles/
    └── vmprober-config/
        ├── defaults/
        │   └── main.yml        # Переменные по умолчанию
        ├── vars/
        │   └── main.yml        # Переменные роли
        ├── handlers/
        │   └── main.yml        # Handlers для сервиса
        ├── tasks/
        │   ├── main.yml        # Главный файл задач
        │   ├── facts.yml       # Сбор фактов и определение архитектуры
        │   ├── prepare.yml     # Подготовка информации о пакете
        │   ├── download.yml    # Скачивание пакета
        │   ├── install.yml     # Установка пакета
        │   ├── configure.yml   # Конфигурация
        │   ├── service.yml     # Управление сервисом
        │   └── cleanup.yml     # Очистка временных файлов
        ├── templates/
        │   └── config.yaml.j2  # Шаблон конфигурации vmprober
        └── README.md           # Документация роли
```

## Быстрый старт

### 1. Настройка inventory

Отредактируйте `inventory.yml` и укажите ваши хосты:

```yaml
all:
  children:
    vmprober:
      hosts:
        vmprober-host-1:
          ansible_host: 192.168.1.100
          ansible_user: ubuntu
        vmprober-host-2:
          ansible_host: 192.168.1.101
          ansible_user: ubuntu
```

### 2. Настройка переменных

Отредактируйте `group_vars/all.yml`:

```yaml
# Версия для установки (тег из GitHub releases)
vmprober_version: "v0.1.0rc1"

# Архитектура (автоматически определяется, можно переопределить)
# vmprober_arch: "amd64"  # или "arm64"

# Настройки проб для нод из inventory
vmprober_tcp_probe_port: 8429  # Порт для TCP проб (порт vmprober)
vmprober_tcp_probe_interval: "10s"
vmprober_tcp_probe_timeout: "5s"
vmprober_icmp_probe_interval: "10s"
vmprober_icmp_probe_timeout: "2s"

# Полная конфигурация vmprober
vmprober_config:
  listen:
    port: 8429
    host: "0.0.0.0"
    tls:
      enabled: false
  pull:
    enabled: true
    path: "/metrics"
    timeout: 10s
  push:
    enabled: true
    endpoints:
      - url: "http://localhost:8480/insert/0/prometheus/api/v1/import"
        headers: {}
        auth:
          type: "none"  # или "basic", "bearer"
    retry:
      max_attempts: 5
      backoff: "exponential"
      initial_delay: "1s"
      max_delay: "60s"
      multiplier: 2.0
    dedup:
      enabled: true
      window: "5m"
      keys: ["job", "instance", "probe", "target", "proto"]
    batch:
      size: 1000
      timeout: "30s"
    remote_write:
      enabled: false
  scheduler:
    concurrent: 100
    rps_limit: 1000
    per_host_cap: 10
    jitter: 0.1
    timeouts:
      tcp: "5s"
      udp: "3s"
      icmp: "2s"
      http: "10s"
      https: "10s"
      dns: "5s"
      grpc: "10s"
    queue_size: 10000
    worker_timeout: "30s"
  targets:
    static: []  # Дополнительные статические пробы (опционально)
```

### 3. Запуск playbook

#### Полное развертывание (рекомендуется)

```bash
ansible-playbook -i inventory.yml playbooks/deploy.yml
```

Этот playbook выполняет:
- Установку vmprober из GitHub releases
- Генерацию конфигурации с автоматическими пробами для всех нод
- Запуск и включение сервиса vmprober

#### Только установка

```bash
ansible-playbook -i inventory.yml playbooks/install.yml
```

#### Только конфигурация

```bash
ansible-playbook -i inventory.yml playbooks/configure.yml
```

#### Только управление сервисом

```bash
ansible-playbook -i inventory.yml playbooks/service.yml
```

## Переменные

### Основные переменные

| Переменная | Описание | По умолчанию |
|-----------|----------|--------------|
| `vmprober_version` | Версия/тег для установки | `v0.1.0rc1` |
| `vmprober_arch` | Архитектура (amd64 или arm64) | Автоматически определяется |
| `vmprober_local_download_dir` | Локальная директория для скачивания | `/tmp/vmprober_downloads` |
| `vmprober_remote_package_dir` | Директория на целевом хосте | `/tmp/vmprober` |

### Переменные управления задачами

| Переменная | Описание | По умолчанию |
|-----------|----------|--------------|
| `vmprober_install` | Выполнять установку | `true` |
| `vmprober_configure` | Выполнять конфигурацию | `true` |
| `vmprober_manage_service` | Управлять сервисом | `true` |
| `vmprober_cleanup` | Выполнять очистку | `true` |

### Переменные проб

| Переменная | Описание | По умолчанию |
|-----------|----------|--------------|
| `vmprober_tcp_probe_port` | Порт для TCP проб | `8429` |
| `vmprober_tcp_probe_interval` | Интервал TCP проб | `"10s"` |
| `vmprober_tcp_probe_timeout` | Таймаут TCP проб | `"5s"` |
| `vmprober_icmp_probe_interval` | Интервал ICMP проб | `"10s"` |
| `vmprober_icmp_probe_timeout` | Таймаут ICMP проб | `"2s"` |

### Конфигурация vmprober

Переменная `vmprober_config` содержит полную конфигурацию vmprober. Роль автоматически генерирует пробы для всех нод из inventory и добавляет их в секцию `targets.static`.

#### Основные секции конфигурации:

- **listen** - настройки HTTP сервера (порт, хост, TLS)
- **pull** - настройки pull режима (Prometheus scrape)
- **push** - настройки push режима (VictoriaMetrics):
  - `endpoints` - список эндпоинтов для отправки метрик
  - `retry` - настройки повторных попыток
  - `dedup` - дедупликация метрик
  - `batch` - настройки батчинга
  - `remote_write` - поддержка Prometheus remote write
- **scheduler** - настройки планировщика задач:
  - `concurrent` - количество одновременных проб
  - `rps_limit` - лимит запросов в секунду
  - `per_host_cap` - максимум проб на хост
  - `jitter` - случайная задержка для распределения нагрузки
  - `timeouts` - таймауты для различных типов проб
  - `queue_size` - размер очереди задач
  - `worker_timeout` - таймаут воркера
- **targets.static** - статические пробы (автоматически дополняются пробами из inventory)

## Автоматическая генерация проб

Роль автоматически создает пробы для всех нод из группы `vmprober` в inventory:

1. **TCP проба** на порт 8429 (или указанный в `vmprober_tcp_probe_port`)
   - Использует IP адрес из `ansible_host` или `inventory_hostname`
   - Метки: `node`, `node_ip`, `probe_type: "tcp"`, `probe_port`

2. **ICMP проба** (ping)
   - Использует IP адрес из `ansible_host` или `inventory_hostname`
   - Метки: `node`, `node_ip`, `probe_type: "icmp"`

Для каждой пробы добавляются метки:
- `node` - имя хоста из inventory
- `node_ip` - IP адрес хоста
- `probe_type` - тип пробы (tcp или icmp)
- `probe_port` - порт (только для TCP проб)

Дополнительные статические пробы можно добавить в `vmprober_config.targets.static`. Поддерживаются все типы проб: TCP, UDP, ICMP, HTTP, HTTPS, DNS, gRPC.

## Примеры использования

### Установка конкретной версии

```bash
ansible-playbook -i inventory.yml playbooks/deploy.yml -e "vmprober_version=v1.0.0"
```

### Только конфигурация без установки

```bash
ansible-playbook -i inventory.yml playbooks/configure.yml
```

### Установка без конфигурации

```bash
ansible-playbook -i inventory.yml playbooks/install.yml
```

### Установка на один хост

```bash
ansible-playbook -i inventory.yml playbooks/deploy.yml --limit vmprober-host-1
```

### Проверка без выполнения (dry-run)

```bash
ansible-playbook -i inventory.yml playbooks/deploy.yml --check --diff
```

### Отключение очистки временных файлов

```bash
ansible-playbook -i inventory.yml playbooks/deploy.yml -e "vmprober_cleanup=false"
```

### Управление сервисом

```bash
# Перезапуск сервиса
ansible-playbook -i inventory.yml playbooks/service.yml

# Остановка сервиса (через переменные)
ansible-playbook -i inventory.yml playbooks/service.yml -e "vmprober_service_state=stopped"
```

### Добавление дополнительных проб

Отредактируйте `group_vars/all.yml` и добавьте пробы в `vmprober_config.targets.static`:

```yaml
vmprober_config:
  targets:
    static:
      - host: "example.com"
        port: 443
        proto: "https"
        interval: "30s"
        timeout: "10s"
        labels:
          service: "example"
        http:
          method: GET
          path: /health
          expected_status_code: 200
      - host: "8.8.8.8"
        port: 53
        proto: "dns"
        interval: "10s"
        timeout: "5s"
        dns:
          query_name: "google.com"
          query_type: A
```

## Структура роли

Роль `vmprober-config` разделена на модульные задачи:

- **facts.yml** - сбор системных фактов и определение архитектуры
- **prepare.yml** - подготовка информации о пакете (имя, URL, пути)
- **download.yml** - скачивание пакета на control node
- **install.yml** - установка пакета на целевые хосты
- **configure.yml** - генерация и применение конфигурации
- **service.yml** - управление сервисом vmprober
- **cleanup.yml** - очистка временных файлов

Каждый модуль может быть включен или отключен через переменные управления задачами.

## Handlers

Роль включает следующие handlers (автоматически вызываются при изменении конфигурации):

- `restart vmprober` - перезапуск сервиса (вызывается при изменении конфигурации)
- `reload vmprober` - перезагрузка конфигурации (если поддерживается)
- `start vmprober` - запуск сервиса
- `stop vmprober` - остановка сервиса

Handlers автоматически вызываются при изменении конфигурационного файла `/etc/vmprober/config.yaml`.

## Требования

- Ansible >= 2.9
- Python 3 на control node
- Python 3 на целевых хостах
- Доступ к интернету с control node для скачивания пакетов из GitHub releases
- SSH доступ к целевым хостам
- Права sudo на целевых хостах
- Система на базе Debian/Ubuntu (для установки .deb пакетов)
- systemd на целевых хостах (для управления сервисом)

## Проверка установки

После выполнения playbook проверьте установку:

```bash
# На целевом хосте
vmprober --version
systemctl status vmprober
cat /etc/vmprober/config.yaml
```

## Устранение неполадок

### Ошибка скачивания пакета

Убедитесь, что:
- Версия указана правильно (существует в GitHub releases)
- Архитектура соответствует целевой системе
- Есть доступ к интернету с control node

### Ошибка установки

Убедитесь, что:
- На целевом хосте установлен apt/dpkg
- Есть права sudo
- Нет конфликтов с предыдущими установками

### Проверка доступных версий

Посмотрите доступные релизы на GitHub:
https://github.com/gdagil/vmprober/releases

### Проблемы с конфигурацией

Проверьте сгенерированную конфигурацию:

```bash
ansible-playbook -i inventory.yml playbooks/configure.yml --check --diff
```

Просмотрите сгенерированный файл на целевом хосте:

```bash
# На целевом хосте
cat /etc/vmprober/config.yaml
```

### Проблемы с сервисом

Проверьте статус сервиса:

```bash
# На целевом хосте
systemctl status vmprober
journalctl -u vmprober -f
```

### Проблемы с архитектурой

Если автоматическое определение архитектуры не работает, укажите явно:

```bash
ansible-playbook -i inventory.yml playbooks/deploy.yml -e "vmprober_arch=amd64"
```

Или в `group_vars/all.yml`:

```yaml
vmprober_arch: "amd64"  # или "arm64"
```

## Конфигурация Ansible

Файл `ansible.cfg` содержит базовые настройки:

```ini
[defaults]
roles_path = ./roles
inventory = inventory.yml
host_key_checking = False
retry_files_enabled = False
```

Эти настройки можно переопределить в вашем локальном `ansible.cfg` или через переменные окружения.
