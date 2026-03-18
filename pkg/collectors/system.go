package collectors

import (
	"bufio"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/zy84338719/dgx-spark-exporter/internal/config"

	"github.com/prometheus/client_golang/prometheus"
)

type SystemCollector struct {
	load1Desc        *prometheus.Desc
	load5Desc        *prometheus.Desc
	load15Desc       *prometheus.Desc
	uptimeDesc       *prometheus.Desc
	procsRunningDesc *prometheus.Desc
	procsBlockedDesc *prometheus.Desc
	ctxtDesc         *prometheus.Desc
	intrDesc         *prometheus.Desc
	fdAllocDesc      *prometheus.Desc
	fdFreeDesc       *prometheus.Desc
	fdMaxDesc        *prometheus.Desc

	logger *slog.Logger
}

func NewSystemCollector(cfg *config.Config, logger *slog.Logger) *SystemCollector {
	return &SystemCollector{
		load1Desc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "load", "average"),
			"System load average",
			[]string{"period"}, nil,
		),
		uptimeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "system", "uptime_seconds"),
			"System uptime in seconds",
			nil, nil,
		),
		procsRunningDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "processes", "running"),
			"Number of processes currently running",
			nil, nil,
		),
		procsBlockedDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "processes", "blocked"),
			"Number of processes currently blocked",
			nil, nil,
		),
		ctxtDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "context", "switches_total"),
			"Total number of context switches",
			nil, nil,
		),
		intrDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "interrupts", "total"),
			"Total number of interrupts",
			nil, nil,
		),
		fdAllocDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "file", "descriptors_allocated"),
			"Number of allocated file descriptors",
			nil, nil,
		),
		fdFreeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "file", "descriptors_free"),
			"Number of free file descriptors",
			nil, nil,
		),
		fdMaxDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "file", "descriptors_max"),
			"Maximum number of file descriptors",
			nil, nil,
		),
		logger: logger.With("collector", "system"),
	}
}

func (c *SystemCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.load1Desc
	ch <- c.load5Desc
	ch <- c.load15Desc
	ch <- c.uptimeDesc
	ch <- c.procsRunningDesc
	ch <- c.procsBlockedDesc
	ch <- c.ctxtDesc
	ch <- c.intrDesc
	ch <- c.fdAllocDesc
	ch <- c.fdFreeDesc
	ch <- c.fdMaxDesc
}

func (c *SystemCollector) Collect(ch chan<- prometheus.Metric) {
	c.collectLoadAvg(ch)
	c.collectUptime(ch)
	c.collectStat(ch)
	c.collectFileDescriptors(ch)
}

func (c *SystemCollector) collectLoadAvg(ch chan<- prometheus.Metric) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		c.logger.Debug("failed to read /proc/loadavg", "error", err)
		return
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	ch <- prometheus.MustNewConstMetric(c.load1Desc, prometheus.GaugeValue, load1, "1m")
	ch <- prometheus.MustNewConstMetric(c.load5Desc, prometheus.GaugeValue, load5, "5m")
	ch <- prometheus.MustNewConstMetric(c.load15Desc, prometheus.GaugeValue, load15, "15m")
}

func (c *SystemCollector) collectUptime(ch chan<- prometheus.Metric) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		c.logger.Debug("failed to read /proc/uptime", "error", err)
		return
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return
	}

	uptime, _ := strconv.ParseFloat(fields[0], 64)
	ch <- prometheus.MustNewConstMetric(c.uptimeDesc, prometheus.GaugeValue, uptime)
}

func (c *SystemCollector) collectStat(ch chan<- prometheus.Metric) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		c.logger.Debug("failed to open /proc/stat", "error", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "ctxt":
			ctxt, _ := strconv.ParseFloat(fields[1], 64)
			ch <- prometheus.MustNewConstMetric(c.ctxtDesc, prometheus.CounterValue, ctxt)

		case "intr":
			intr, _ := strconv.ParseFloat(fields[1], 64)
			ch <- prometheus.MustNewConstMetric(c.intrDesc, prometheus.CounterValue, intr)

		case "procs_running":
			running, _ := strconv.ParseFloat(fields[1], 64)
			ch <- prometheus.MustNewConstMetric(c.procsRunningDesc, prometheus.GaugeValue, running)

		case "procs_blocked":
			blocked, _ := strconv.ParseFloat(fields[1], 64)
			ch <- prometheus.MustNewConstMetric(c.procsBlockedDesc, prometheus.GaugeValue, blocked)
		}
	}
}

func (c *SystemCollector) collectFileDescriptors(ch chan<- prometheus.Metric) {
	data, err := os.ReadFile("/proc/sys/fs/file-nr")
	if err != nil {
		c.logger.Debug("failed to read /proc/sys/fs/file-nr", "error", err)
		return
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return
	}

	allocated, _ := strconv.ParseFloat(fields[0], 64)
	free, _ := strconv.ParseFloat(fields[1], 64)
	max, _ := strconv.ParseFloat(fields[2], 64)

	ch <- prometheus.MustNewConstMetric(c.fdAllocDesc, prometheus.GaugeValue, allocated)
	ch <- prometheus.MustNewConstMetric(c.fdFreeDesc, prometheus.GaugeValue, free)
	ch <- prometheus.MustNewConstMetric(c.fdMaxDesc, prometheus.GaugeValue, max)
}
