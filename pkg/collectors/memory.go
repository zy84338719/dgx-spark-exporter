package collectors

import (
	"log/slog"

	"github.com/zy84338719/dgx-spark-exporter/internal/config"
	"github.com/zy84338719/dgx-spark-exporter/pkg/collectors/utils"

	"github.com/prometheus/client_golang/prometheus"
)

type MemoryCollector struct {
	totalDesc     *prometheus.Desc
	usedDesc      *prometheus.Desc
	availableDesc *prometheus.Desc

	logger *slog.Logger
}

func NewMemoryCollector(cfg *config.Config, logger *slog.Logger) *MemoryCollector {
	return &MemoryCollector{
		totalDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "memory", "total_bytes"),
			"Total physical RAM in bytes",
			nil, nil,
		),
		usedDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "memory", "used_bytes"),
			"Used RAM in bytes (total - available)",
			nil, nil,
		),
		availableDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "memory", "available_bytes"),
			"Available RAM in bytes",
			nil, nil,
		),
		logger: logger.With("collector", "memory"),
	}
}

func (c *MemoryCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalDesc
	ch <- c.usedDesc
	ch <- c.availableDesc
}

func (c *MemoryCollector) Collect(ch chan<- prometheus.Metric) {
	memInfo, err := utils.ParseKeyValueFileUint64("/proc/meminfo")
	if err != nil {
		c.logger.Debug("failed to read /proc/meminfo", "error", err)
		return
	}

	totalKB := memInfo["MemTotal"]
	freeKB := memInfo["MemFree"]
	buffersKB := memInfo["Buffers"]
	cachedKB := memInfo["Cached"]
	availableKB := memInfo["MemAvailable"]

	totalBytes := float64(totalKB) * 1024

	var availableBytes float64
	if availableKB > 0 {
		availableBytes = float64(availableKB) * 1024
	} else {
		availableBytes = float64(freeKB+buffersKB+cachedKB) * 1024
	}

	usedBytes := totalBytes - availableBytes
	if usedBytes < 0 {
		usedBytes = totalBytes - float64(freeKB)*1024
	}

	ch <- prometheus.MustNewConstMetric(c.totalDesc, prometheus.GaugeValue, totalBytes)
	ch <- prometheus.MustNewConstMetric(c.usedDesc, prometheus.GaugeValue, usedBytes)
	ch <- prometheus.MustNewConstMetric(c.availableDesc, prometheus.GaugeValue, availableBytes)
}
