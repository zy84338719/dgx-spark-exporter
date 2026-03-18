package collectors

import (
	"log/slog"

	"github.com/zy84338719/dgx-spark-exporter/internal/config"
	"github.com/zy84338719/dgx-spark-exporter/pkg/collectors/utils"

	"github.com/prometheus/client_golang/prometheus"
)

type SwapCollector struct {
	totalDesc *prometheus.Desc
	usedDesc  *prometheus.Desc
	freeDesc  *prometheus.Desc

	logger *slog.Logger
}

func NewSwapCollector(cfg *config.Config, logger *slog.Logger) *SwapCollector {
	return &SwapCollector{
		totalDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "swap", "total_bytes"),
			"Total swap space in bytes",
			nil, nil,
		),
		usedDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "swap", "used_bytes"),
			"Used swap space in bytes",
			nil, nil,
		),
		freeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "swap", "free_bytes"),
			"Free swap space in bytes",
			nil, nil,
		),
		logger: logger.With("collector", "swap"),
	}
}

func (c *SwapCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalDesc
	ch <- c.usedDesc
	ch <- c.freeDesc
}

func (c *SwapCollector) Collect(ch chan<- prometheus.Metric) {
	memInfo, err := utils.ParseKeyValueFileUint64("/proc/meminfo")
	if err != nil {
		c.logger.Debug("failed to read /proc/meminfo", "error", err)
		return
	}

	swapTotalKB := memInfo["SwapTotal"]
	swapFreeKB := memInfo["SwapFree"]
	swapCachedKB := memInfo["SwapCached"]

	totalBytes := float64(swapTotalKB) * 1024
	freeBytes := float64(swapFreeKB) * 1024

	var usedBytes float64
	if swapCachedKB > 0 {
		usedBytes = float64(swapTotalKB-swapFreeKB-swapCachedKB) * 1024
	} else {
		usedBytes = totalBytes - freeBytes
	}

	if usedBytes < 0 {
		usedBytes = 0
	}

	ch <- prometheus.MustNewConstMetric(c.totalDesc, prometheus.GaugeValue, totalBytes)
	ch <- prometheus.MustNewConstMetric(c.usedDesc, prometheus.GaugeValue, usedBytes)
	ch <- prometheus.MustNewConstMetric(c.freeDesc, prometheus.GaugeValue, freeBytes)
}
