# vmprober-config Role

Ansible роль для генерации конфигурации vmprober с автоматическими пробами для всех нод из inventory.

## Описание

Эта роль автоматически генерирует конфигурацию vmprober, включая:
- TCP пробы на порт 8429 (порт vmprober) для всех нод из группы `vmprober` в inventory
- ICMP пробы (ping) для всех нод из группы `vmprober` в inventory
- Дополнительные статические пробы, определенные в `vmprober_config.targets.static`

## Использование

### Базовое использование

1. Убедитесь, что в `inventory.yml` определена группа `vmprober` с хостами:

```yaml
all:
  children:
    vmprober:
      hosts:
        vmprober-host-1:
          ansible_host: 192.168.1.100
        vmprober-host-2:
          ansible_host: 192.168.1.101
```

2. Настройте переменные в `group_vars/all.yml`:

```yaml
# Настройки проб для нод из inventory
vmprober_tcp_probe_port: 8429
vmprober_tcp_probe_interval: "10s"
vmprober_tcp_probe_timeout: "5s"
vmprober_icmp_probe_interval: "10s"
vmprober_icmp_probe_timeout: "2s"

# Полная конфигурация vmprober (опционально)
vmprober_config:
  push:
    enabled: true
    endpoints:
      - url: "http://localhost:8480/insert/0/prometheus/api/v1/import"
        headers: {}
        auth:
          type: "none"
  # ... другие настройки
```

3. Запустите playbook:

```bash
ansible-playbook -i inventory.yml configure-vmprober.yml
```

### Переменные роли

#### Переменные для проб inventory нод

- `vmprober_tcp_probe_port` (по умолчанию: `8429`) - порт для TCP проб
- `vmprober_tcp_probe_interval` (по умолчанию: `"10s"`) - интервал TCP проб
- `vmprober_tcp_probe_timeout` (по умолчанию: `"5s"`) - таймаут TCP проб
- `vmprober_icmp_probe_interval` (по умолчанию: `"10s"`) - интервал ICMP проб
- `vmprober_icmp_probe_timeout` (по умолчанию: `"2s"`) - таймаут ICMP проб

#### Переменная полной конфигурации

- `vmprober_config` - словарь с полной конфигурацией vmprober (опционально)

## Генерируемые пробы

Для каждой ноды из группы `vmprober` в inventory автоматически создаются:

1. **TCP проба** на порт 8429 (или указанный в `vmprober_tcp_probe_port`):
   - Использует IP адрес из `ansible_host` или `inventory_hostname`
   - Метки: `node`, `node_ip`, `probe_type: "tcp"`, `probe_port`

2. **ICMP проба** (ping):
   - Использует IP адрес из `ansible_host` или `inventory_hostname`
   - Метки: `node`, `node_ip`, `probe_type: "icmp"`

## Пример сгенерированной конфигурации

Для двух нод в inventory будет сгенерировано:

```yaml
targets:
  static:
    # TCP probe for vmprober-host-1 (192.168.1.100)
    - host: "192.168.1.100"
      port: 8429
      proto: "tcp"
      interval: "10s"
      timeout: "5s"
      labels:
        node: "vmprober-host-1"
        node_ip: "192.168.1.100"
        probe_type: "tcp"
        probe_port: "8429"

    # ICMP probe for vmprober-host-1 (192.168.1.100)
    - host: "192.168.1.100"
      proto: "icmp"
      interval: "10s"
      timeout: "2s"
      labels:
        node: "vmprober-host-1"
        node_ip: "192.168.1.100"
        probe_type: "icmp"

    # TCP probe for vmprober-host-2 (192.168.1.101)
    - host: "192.168.1.101"
      port: 8429
      proto: "tcp"
      interval: "10s"
      timeout: "5s"
      labels:
        node: "vmprober-host-2"
        node_ip: "192.168.1.101"
        probe_type: "tcp"
        probe_port: "8429"

    # ICMP probe for vmprober-host-2 (192.168.1.101)
    - host: "192.168.1.101"
      proto: "icmp"
      interval: "10s"
      timeout: "2s"
      labels:
        node: "vmprober-host-2"
        node_ip: "192.168.1.101"
        probe_type: "icmp"
```

## Дополнительные статические пробы

Вы можете добавить дополнительные статические пробы через `vmprober_config.targets.static`:

```yaml
vmprober_config:
  targets:
    static:
      - host: "example.com"
        port: 80
        proto: "tcp"
        interval: "10s"
        timeout: "5s"
        labels:
          service: "example"
```

Эти пробы будут добавлены после автоматически сгенерированных проб для нод из inventory.

## Зависимости

- Роль требует, чтобы vmprober был установлен на целевых хостах
- Роль использует handler для перезапуска сервиса vmprober после изменения конфигурации

## Файлы роли

- `tasks/main.yml` - основные задачи роли
- `templates/config.yaml.j2` - шаблон конфигурации vmprober
- `defaults/main.yml` - переменные по умолчанию
