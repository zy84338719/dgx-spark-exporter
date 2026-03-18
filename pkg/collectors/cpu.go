package collectors

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/zy84338719/dgx-spark-exporter/internal/config"
	"github.com/zy84338719/dgx-spark-exporter/pkg/collectors/utils"

	"github.com/prometheus/client_golang/prometheus"
)

type CPUCollector struct {
	usageDesc *prometheus.Desc
	tempDesc  *prometheus.Desc
	freqDesc  *prometheus.Desc
	timeDesc  *prometheus.Desc
	coresDesc *prometheus.Desc

	mu           sync.Mutex
	prevStats    map[string]uint64
	thermalZones int
	maxCPUs      int
	logger       *slog.Logger
}

func NewCPUCollector(cfg *config.Config, logger *slog.Logger) *CPUCollector {
	return &CPUCollector{
		usageDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "cpu", "usage_ratio"),
			"CPU usage ratio (0-1)",
			nil, nil,
		),
		tempDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "cpu", "temperature_celsius"),
			"CPU temperature in degrees Celsius",
			[]string{"zone", "type"}, nil,
		),
		freqDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "cpu", "frequency_hertz"),
			"Average CPU core frequency in Hz",
			nil, nil,
		),
		timeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "cpu", "time_seconds_total"),
			"Total CPU time spent in each mode",
			[]string{"mode"}, nil,
		),
		coresDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "cpu", "cores"),
			"Number of CPU cores",
			nil, nil,
		),
		prevStats:    make(map[string]uint64),
		thermalZones: cfg.ThermalZoneCount,
		maxCPUs:      cfg.MaxCPUCount,
		logger:       logger.With("collector", "cpu"),
	}
}

func (c *CPUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.usageDesc
	ch <- c.tempDesc
	ch <- c.freqDesc
	ch <- c.timeDesc
	ch <- c.coresDesc
}

func (c *CPUCollector) Collect(ch chan<- prometheus.Metric) {
	c.collectUsage(ch)
	c.collectTemperatures(ch)
	c.collectFrequency(ch)
	c.collectCPUTime(ch)
	c.collectCores(ch)
}

func (c *CPUCollector) collectUsage(ch chan<- prometheus.Metric) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		c.logger.Debug("failed to open /proc/stat", "error", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return
	}

	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 8 || fields[0] != "cpu" {
		return
	}

	user := parseUintOrZero(fields[1])
	nice := parseUintOrZero(fields[2])
	system := parseUintOrZero(fields[3])
	idle := parseUintOrZero(fields[4])
	iowait := parseUintOrZero(fields[5])
	irq := parseUintOrZero(fields[6])
	softirq := parseUintOrZero(fields[7])

	total := user + nice + system + idle + iowait + irq + softirq
	idleTotal := idle + iowait

	c.mu.Lock()
	prevTotal := c.prevStats["total"]
	prevIdle := c.prevStats["idle"]
	c.prevStats["total"] = total
	c.prevStats["idle"] = idleTotal
	c.mu.Unlock()

	if prevTotal == 0 {
		ch <- prometheus.MustNewConstMetric(c.usageDesc, prometheus.GaugeValue, 0)
		return
	}

	totalDelta := total - prevTotal
	idleDelta := idleTotal - prevIdle

	if totalDelta == 0 {
		ch <- prometheus.MustNewConstMetric(c.usageDesc, prometheus.GaugeValue, 0)
		return
	}

	usage := float64(totalDelta-idleDelta) / float64(totalDelta)
	ch <- prometheus.MustNewConstMetric(c.usageDesc, prometheus.GaugeValue, usage)
}

func (c *CPUCollector) collectTemperatures(ch chan<- prometheus.Metric) {
	cpuZoneFound := false

	for i := 0; i < c.thermalZones; i++ {
		typePath := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/type", i)
		zoneType, err := utils.ReadFileString(typePath)
		if err != nil {
			continue
		}

		tempPath := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/temp", i)
		millideg, err := utils.ReadFileFloat(tempPath)
		if err != nil {
			continue
		}

		temp := millideg / 1000.0
		zoneStr := fmt.Sprintf("%d", i)
		ch <- prometheus.MustNewConstMetric(c.tempDesc, prometheus.GaugeValue, temp, zoneStr, zoneType)

		zoneTypeLower := toLower(zoneType)
		if contains(zoneTypeLower, "cpu") || contains(zoneTypeLower, "soc") || contains(zoneTypeLower, "acpi") {
			cpuZoneFound = true
		}
	}

	if !cpuZoneFound {
		for i := 0; i < c.thermalZones; i++ {
			tempPath := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/temp", i)
			if millideg, err := utils.ReadFileFloat(tempPath); err == nil {
				temp := millideg / 1000.0
				zoneStr := fmt.Sprintf("%d", i)
				typePath := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/type", i)
				zoneType, _ := utils.ReadFileString(typePath)
				if zoneType == "" {
					zoneType = "unknown"
				}
				ch <- prometheus.MustNewConstMetric(c.tempDesc, prometheus.GaugeValue, temp, zoneStr, zoneType)
				break
			}
		}
	}
}

func (c *CPUCollector) collectFrequency(ch chan<- prometheus.Metric) {
	var totalFreq float64
	count := 0

	for i := 0; i < c.maxCPUs; i++ {
		path := fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/scaling_cur_freq", i)
		freqKHz, err := utils.ReadFileFloat(path)
		if err != nil {
			if i == 0 {
				return
			}
			break
		}
		totalFreq += freqKHz
		count++
	}

	if count == 0 {
		return
	}

	avgFreq := totalFreq / float64(count) * 1000.0
	ch <- prometheus.MustNewConstMetric(c.freqDesc, prometheus.GaugeValue, avgFreq)
}

func (c *CPUCollector) collectCPUTime(ch chan<- prometheus.Metric) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		c.logger.Debug("failed to open /proc/stat", "error", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return
	}

	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 8 || fields[0] != "cpu" {
		return
	}

	modes := []string{"user", "nice", "system", "idle", "iowait", "irq", "softirq"}
	for i, mode := range modes {
		if i+1 < len(fields) {
			value := parseUintOrZero(fields[i+1])
			ch <- prometheus.MustNewConstMetric(c.timeDesc, prometheus.CounterValue, float64(value)/100.0, mode)
		}
	}

	if len(fields) > 8 {
		steal := parseUintOrZero(fields[8])
		ch <- prometheus.MustNewConstMetric(c.timeDesc, prometheus.CounterValue, float64(steal)/100.0, "steal")
	}
	if len(fields) > 9 {
		guest := parseUintOrZero(fields[9])
		ch <- prometheus.MustNewConstMetric(c.timeDesc, prometheus.CounterValue, float64(guest)/100.0, "guest")
	}
	if len(fields) > 10 {
		guestNice := parseUintOrZero(fields[10])
		ch <- prometheus.MustNewConstMetric(c.timeDesc, prometheus.CounterValue, float64(guestNice)/100.0, "guest_nice")
	}
}

func (c *CPUCollector) collectCores(ch chan<- prometheus.Metric) {
	count := 0
	for i := 0; i < c.maxCPUs; i++ {
		path := fmt.Sprintf("/sys/devices/system/cpu/cpu%d", i)
		if _, err := os.Stat(path); err == nil {
			count++
		} else {
			break
		}
	}

	if count > 0 {
		ch <- prometheus.MustNewConstMetric(c.coresDesc, prometheus.GaugeValue, float64(count))
	}
}

func parseUintOrZero(s string) uint64 {
	var result uint64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + uint64(c-'0')
		} else {
			break
		}
	}
	return result
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
