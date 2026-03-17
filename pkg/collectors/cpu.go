package collectors

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/zy84338719/dgx-spark-exporter/internal/config"
	"github.com/zy84338719/dgx-spark-exporter/pkg/collectors/utils"

	"github.com/prometheus/client_golang/prometheus"
)

type CPUCollector struct {
	usageDesc *prometheus.Desc
	tempDesc  *prometheus.Desc
	freqDesc  *prometheus.Desc

	mu           sync.Mutex
	prevIdle     uint64
	prevTotal    uint64
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
			nil, nil,
		),
		freqDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "cpu", "frequency_hertz"),
			"Average CPU core frequency in Hz",
			nil, nil,
		),
		thermalZones: cfg.ThermalZoneCount,
		maxCPUs:      cfg.MaxCPUCount,
		logger:       logger.With("collector", "cpu"),
	}
}

func (c *CPUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.usageDesc
	ch <- c.tempDesc
	ch <- c.freqDesc
}

func (c *CPUCollector) Collect(ch chan<- prometheus.Metric) {
	if usage, err := c.readUsage(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.usageDesc, prometheus.GaugeValue, usage)
	} else {
		c.logger.Debug("failed to read CPU usage", "error", err)
	}

	if temp, err := c.readTemperature(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.tempDesc, prometheus.GaugeValue, temp)
	} else {
		c.logger.Debug("failed to read CPU temperature", "error", err)
	}

	if freq, err := c.readFrequency(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.freqDesc, prometheus.GaugeValue, freq)
	} else {
		c.logger.Debug("failed to read CPU frequency", "error", err)
	}
}

func (c *CPUCollector) readUsage() (float64, error) {
	statMap, err := utils.ParseKeyValueFileUint64("/proc/stat")
	if err != nil {
		return 0, err
	}

	cpuLine, err := utils.ReadFileString("/proc/stat")
	if err != nil {
		return 0, err
	}

	_ = statMap

	fields := splitFields(cpuLine)
	if len(fields) < 8 {
		return 0, fmt.Errorf("invalid /proc/stat format")
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
	prevTotal := c.prevTotal
	prevIdle := c.prevIdle
	c.prevTotal = total
	c.prevIdle = idleTotal
	c.mu.Unlock()

	if prevTotal == 0 {
		return 0, nil
	}

	totalDelta := total - prevTotal
	idleDelta := idleTotal - prevIdle

	if totalDelta == 0 {
		return 0, nil
	}

	usage := float64(totalDelta-idleDelta) / float64(totalDelta)
	return usage, nil
}

func (c *CPUCollector) readTemperature() (float64, error) {
	for i := 0; i < c.thermalZones; i++ {
		typePath := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/type", i)
		zoneType, err := utils.ReadFileString(typePath)
		if err != nil {
			continue
		}

		zoneTypeLower := toLower(zoneType)
		if contains(zoneTypeLower, "cpu") || contains(zoneTypeLower, "soc") {
			tempPath := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/temp", i)
			return c.readThermalTemp(tempPath)
		}
	}

	return c.readThermalTemp("/sys/class/thermal/thermal_zone0/temp")
}

func (c *CPUCollector) readThermalTemp(path string) (float64, error) {
	millideg, err := utils.ReadFileFloat(path)
	if err != nil {
		return 0, err
	}
	return millideg / 1000.0, nil
}

func (c *CPUCollector) readFrequency() (float64, error) {
	var totalFreq float64
	count := 0

	for i := 0; i < c.maxCPUs; i++ {
		path := fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/scaling_cur_freq", i)
		freqKHz, err := utils.ReadFileFloat(path)
		if err != nil {
			if i == 0 {
				return 0, fmt.Errorf("no cpufreq support")
			}
			break
		}
		totalFreq += freqKHz
		count++
	}

	if count == 0 {
		return 0, fmt.Errorf("no CPU frequency data")
	}

	return totalFreq / float64(count) * 1000.0, nil
}

func splitFields(s string) []string {
	var result []string
	start := 0
	inSpace := false

	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			if !inSpace && i > start {
				result = append(result, s[start:i])
			}
			inSpace = true
			start = i + 1
		} else {
			inSpace = false
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
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
