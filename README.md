# DGX Spark Exporter

[![Go Report Card](https://goreportcard.com/badge/github.com/zy84338719/dgx-spark-exporter)](https://goreportcard.com/report/github.com/zy84338719/dgx-spark-exporter)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](LICENSE)

A [Prometheus](https://prometheus.io) metrics exporter for [NVIDIA DGX Spark](https://www.nvidia.com/en-us/products/workstations/dgx-spark/) systems.

## Features

- CPU usage, temperature, and frequency monitoring
- GPU metrics via nvidia-smi (utilization, temperature, power, frequency)
- Memory usage tracking
- Disk I/O and storage capacity metrics
- Network traffic statistics
- Low overhead, single binary deployment
- Standard Prometheus exporter conventions

## Endpoints

| Endpoint | Description |
|----------|-------------|
| `/` | Landing page with links |
| `/metrics` | Prometheus metrics |
| `/health` | Health check (returns `OK`) |
| `/ready` | Readiness check (returns `Ready`) |
| `/version` | Version information (JSON) |

## Metrics

All metrics are prefixed with `dgx_spark_` namespace.

### CPU Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dgx_spark_cpu_usage_ratio` | Gauge | | CPU usage ratio (0-1) |
| `dgx_spark_cpu_temperature_celsius` | Gauge | zone, type | CPU temperature in Celsius |
| `dgx_spark_cpu_frequency_hertz` | Gauge | | Average CPU core frequency in Hz |
| `dgx_spark_cpu_time_seconds_total` | Counter | mode | Total CPU time spent in each mode |
| `dgx_spark_cpu_cores` | Gauge | | Number of CPU cores |

### GPU Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `dgx_spark_gpu_utilization_ratio` | Gauge | GPU utilization ratio (0-1) |
| `dgx_spark_gpu_temperature_celsius` | Gauge | GPU temperature in Celsius |
| `dgx_spark_gpu_frequency_hertz` | Gauge | GPU graphics clock frequency in Hz |
| `dgx_spark_gpu_power_watts` | Gauge | GPU power consumption in Watts |

### Memory Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `dgx_spark_memory_total_bytes` | Gauge | Total physical RAM in bytes |
| `dgx_spark_memory_used_bytes` | Gauge | Used RAM in bytes |
| `dgx_spark_memory_available_bytes` | Gauge | Available RAM in bytes |

### Disk Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dgx_spark_disk_reads_completed_total` | Counter | device | Total completed disk read operations |
| `dgx_spark_disk_writes_completed_total` | Counter | device | Total completed disk write operations |
| `dgx_spark_disk_read_bytes_total` | Counter | device | Total bytes read from disk |
| `dgx_spark_disk_written_bytes_total` | Counter | device | Total bytes written to disk |

### Filesystem Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dgx_spark_filesystem_size_bytes` | Gauge | mountpoint | Total filesystem size in bytes |
| `dgx_spark_filesystem_avail_bytes` | Gauge | mountpoint | Available filesystem space in bytes |
| `dgx_spark_filesystem_used_ratio` | Gauge | mountpoint | Used filesystem ratio (0-1) |

### Network Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dgx_spark_network_receive_bytes_total` | Counter | interface | Total bytes received |
| `dgx_spark_network_transmit_bytes_total` | Counter | interface | Total bytes transmitted |
| `dgx_spark_network_receive_packets_total` | Counter | interface | Total packets received |
| `dgx_spark_network_transmit_packets_total` | Counter | interface | Total packets transmitted |
| `dgx_spark_network_receive_errors_total` | Counter | interface | Total receive errors |
| `dgx_spark_network_transmit_errors_total` | Counter | interface | Total transmit errors |
| `dgx_spark_network_receive_dropped_total` | Counter | interface | Total receive dropped |
| `dgx_spark_network_transmit_dropped_total` | Counter | interface | Total transmit dropped |

### System Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dgx_spark_load_average` | Gauge | period | System load average (1m, 5m, 15m) |
| `dgx_spark_system_uptime_seconds` | Gauge | | System uptime in seconds |
| `dgx_spark_processes_running` | Gauge | | Number of running processes |
| `dgx_spark_processes_blocked` | Gauge | | Number of blocked processes |
| `dgx_spark_context_switches_total` | Counter | | Total context switches |
| `dgx_spark_interrupts_total` | Counter | | Total interrupts |
| `dgx_spark_file_descriptors_allocated` | Gauge | | Allocated file descriptors |
| `dgx_spark_file_descriptors_free` | Gauge | | Free file descriptors |
| `dgx_spark_file_descriptors_max` | Gauge | | Maximum file descriptors |

### Swap Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `dgx_spark_swap_total_bytes` | Gauge | Total swap space in bytes |
| `dgx_spark_swap_used_bytes` | Gauge | Used swap space in bytes |
| `dgx_spark_swap_free_bytes` | Gauge | Free swap space in bytes |

### Power Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dgx_spark_system_power_watts` | Gauge | source | System power consumption in Watts |
| `dgx_spark_cpu_package_power_watts` | Gauge | package | CPU package power in Watts (RAPL) |
| `dgx_spark_dram_power_watts` | Gauge | package | DRAM power in Watts (RAPL) |

### Storage Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dgx_spark_nvme_temperature_celsius` | Gauge | device | NVMe temperature in Celsius |
| `dgx_spark_disk_temperature_celsius` | Gauge | device | Disk temperature in Celsius |

### Build Info

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dgx_spark_exporter_build_info` | Gauge | version, revision, branch, go_version | Build information |

## Quick Start

### Build

```bash
make build
```

### Run

```bash
./dgx-spark-exporter
```

Metrics available at `http://localhost:9876/metrics`

## Configuration

### Command-line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-listen` | `:9876` | Listen address |
| `-metrics-path` | `/metrics` | Metrics endpoint path |
| `-log-level` | `info` | Log level (debug, info, warn, error) |
| `-scrape-timeout` | `10s` | GPU scrape timeout |
| `-interfaces` | (auto) | Network interfaces to monitor |
| `-collectors` | `cpu,gpu,memory,disk,network,system,swap,power,storage` | Enabled collectors |
| `-root-mount` | `/` | Root mount for storage metrics |
| `-thermal-zones` | `10` | Thermal zones to scan |
| `-max-cpus` | `256` | Max CPU cores to scan |
| `-power-source` | `auto` | Power monitoring source (auto, rapl, acpi, estimated) |
| `-powercap-path` | `/sys/class/powercap` | Path to powercap interface |

### Environment Variables

| Variable | Flag |
|----------|------|
| `LISTEN_ADDR` | `-listen` |
| `METRICS_PATH` | `-metrics-path` |
| `LOG_LEVEL` | `-log-level` |
| `NETWORK_INTERFACES` | `-interfaces` |
| `COLLECTORS` | `-collectors` |

## Installation

### Systemd Service (Recommended)

```bash
# Install and start service
sudo make service-install

# Check status
make service-status

# View logs
journalctl -u dgx-spark-exporter -f
```

### Service Management

```bash
sudo make service-start      # Start service
sudo make service-stop       # Stop service
sudo make service-restart    # Restart service
make service-status          # Check status
sudo make service-uninstall  # Remove service
```

### Manual Install

```bash
sudo cp dgx-spark-exporter /usr/local/bin/
sudo cp deploy/dgx-spark-exporter.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now dgx-spark-exporter
```

## Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'dgx_spark'
    scrape_interval: 5s
    static_configs:
      - targets: ['spark1:9876', 'spark2:9876']
```

## Grafana Dashboard

Import `deploy/grafana-dashboard.json` into Grafana for ready-to-use visualizations.

![Dashboard Preview](dashboard.png)

## Project Structure

```
├── cmd/dgx-spark-exporter/  # Application entrypoint
├── pkg/collectors/          # Metrics collectors
├── internal/                # Internal packages
│   ├── config/              # Configuration
│   └── logger/              # Logging
├── deploy/                  # Deployment files
├── Makefile
├── go.mod
└── README.md
```

## Data Sources

| Metric | Source |
|--------|--------|
| CPU usage | `/proc/stat` |
| CPU temperature | `/sys/class/thermal/thermal_zone*/` |
| CPU frequency | `/sys/devices/system/cpu/cpu*/cpufreq/` |
| CPU time | `/proc/stat` |
| GPU metrics | `nvidia-smi` |
| Memory | `/proc/meminfo` |
| Swap | `/proc/meminfo` |
| Disk I/O | `/proc/diskstats` |
| Storage | `statfs()` |
| Network | `/sys/class/net/*/statistics/` |
| System load | `/proc/loadavg` |
| System uptime | `/proc/uptime` |
| Processes | `/proc/stat` |
| File descriptors | `/proc/sys/fs/file-nr` |
| System power | `/sys/class/powercap/`, `nvidia-smi`, estimated |
| NVMe temperature | `/sys/class/hwmon/` |

## Development

```bash
make fmt      # Format code
make vet      # Run go vet
make test     # Run tests
make lint     # Run golangci-lint
```

## License

[BSD 3-Clause](LICENSE)

## Contributing

Contributions welcome! Please open an issue or submit a pull request.
