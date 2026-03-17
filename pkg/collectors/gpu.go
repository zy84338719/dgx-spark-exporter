package collectors

import (
	"context"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/zy84338719/dgx-spark-exporter/internal/config"

	"github.com/prometheus/client_golang/prometheus"
)

type GPUCollector struct {
	utilizationDesc *prometheus.Desc
	tempDesc        *prometheus.Desc
	freqDesc        *prometheus.Desc
	powerDesc       *prometheus.Desc

	timeout time.Duration
	logger  *slog.Logger
}

func NewGPUCollector(cfg *config.Config, logger *slog.Logger) *GPUCollector {
	return &GPUCollector{
		utilizationDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "gpu", "utilization_ratio"),
			"GPU utilization ratio (0-1)",
			nil, nil,
		),
		tempDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "gpu", "temperature_celsius"),
			"GPU temperature in degrees Celsius",
			nil, nil,
		),
		freqDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "gpu", "frequency_hertz"),
			"GPU graphics clock frequency in Hz",
			nil, nil,
		),
		powerDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "gpu", "power_watts"),
			"GPU power consumption in Watts",
			nil, nil,
		),
		timeout: cfg.ScrapeTimeout,
		logger:  logger.With("collector", "gpu"),
	}
}

func (c *GPUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.utilizationDesc
	ch <- c.tempDesc
	ch <- c.freqDesc
	ch <- c.powerDesc
}

func (c *GPUCollector) Collect(ch chan<- prometheus.Metric) {
	metrics, err := c.collect()
	if err != nil {
		c.logger.Debug("failed to collect GPU metrics", "error", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.utilizationDesc, prometheus.GaugeValue, metrics.Utilization)
	ch <- prometheus.MustNewConstMetric(c.tempDesc, prometheus.GaugeValue, metrics.Temperature)
	ch <- prometheus.MustNewConstMetric(c.freqDesc, prometheus.GaugeValue, metrics.Frequency)
	ch <- prometheus.MustNewConstMetric(c.powerDesc, prometheus.GaugeValue, metrics.Power)
}

type GPUMetrics struct {
	Utilization float64
	Temperature float64
	Power       float64
	Frequency   float64
}

func (c *GPUCollector) collect() (*GPUMetrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=utilization.gpu,temperature.gpu,power.draw,clocks.current.graphics",
		"--format=csv,noheader,nounits")

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return nil, err
	}

	fields := strings.Split(lines[0], ",")
	if len(fields) < 4 {
		return nil, err
	}

	return &GPUMetrics{
		Utilization: parseNvidiaFloat(fields[0]) / 100.0,
		Temperature: parseNvidiaFloat(fields[1]),
		Power:       parseNvidiaFloat(fields[2]),
		Frequency:   parseNvidiaFloat(fields[3]) * 1e6,
	}, nil
}

func parseNvidiaFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "[N/A]" || s == "N/A" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
