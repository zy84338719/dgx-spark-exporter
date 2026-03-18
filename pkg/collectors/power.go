package collectors

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zy84338719/dgx-spark-exporter/internal/config"

	"github.com/prometheus/client_golang/prometheus"
)

type PowerCollector struct {
	systemPowerDesc  *prometheus.Desc
	packagePowerDesc *prometheus.Desc
	dramPowerDesc    *prometheus.Desc

	powerSource  string
	powercapPath string
	timeout      time.Duration
	logger       *slog.Logger

	mu               sync.Mutex
	prevEnergyValues map[string]uint64
	prevEnergyTime   time.Time
}

func NewPowerCollector(cfg *config.Config, logger *slog.Logger) *PowerCollector {
	return &PowerCollector{
		systemPowerDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "system", "power_watts"),
			"System power consumption in Watts",
			[]string{"source"}, nil,
		),
		packagePowerDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "cpu", "package_power_watts"),
			"CPU package power consumption in Watts",
			[]string{"package"}, nil,
		),
		dramPowerDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "dram", "power_watts"),
			"DRAM power consumption in Watts",
			[]string{"package"}, nil,
		),
		powerSource:      cfg.PowerSource,
		powercapPath:     cfg.PowercapPath,
		timeout:          cfg.ScrapeTimeout,
		logger:           logger.With("collector", "power"),
		prevEnergyValues: make(map[string]uint64),
		prevEnergyTime:   time.Time{},
	}
}

func (c *PowerCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.systemPowerDesc
	ch <- c.packagePowerDesc
	ch <- c.dramPowerDesc
}

func (c *PowerCollector) Collect(ch chan<- prometheus.Metric) {
	var power float64
	var source string
	var err error

	switch c.powerSource {
	case "rapl":
		power, source, err = c.collectRAPL(ch)
	case "acpi":
		power, source, err = c.collectACPI()
	case "estimated":
		power, source, err = c.collectEstimated()
	default:
		power, source, err = c.collectAuto(ch)
	}

	if err != nil {
		c.logger.Debug("failed to collect power", "error", err, "source", source)
		return
	}

	if power > 0 {
		ch <- prometheus.MustNewConstMetric(c.systemPowerDesc, prometheus.GaugeValue, power, source)
	}
}

func (c *PowerCollector) collectAuto(ch chan<- prometheus.Metric) (float64, string, error) {
	if power, source, err := c.collectRAPL(ch); err == nil && power > 0 {
		return power, source, nil
	}

	if power, source, err := c.collectACPI(); err == nil && power > 0 {
		return power, source, nil
	}

	return c.collectEstimated()
}

func (c *PowerCollector) collectRAPL(ch chan<- prometheus.Metric) (float64, string, error) {
	intelRaplPath := filepath.Join(c.powercapPath, "intel-rapl:0")
	if _, err := os.Stat(intelRaplPath); os.IsNotExist(err) {
		return 0, "rapl", fmt.Errorf("RAPL not available")
	}

	totalPower := 0.0

	packageIdx := 0
	for {
		raplPath := filepath.Join(c.powercapPath, fmt.Sprintf("intel-rapl:%d", packageIdx))
		if _, err := os.Stat(raplPath); os.IsNotExist(err) {
			break
		}

		pkgEnergy, err := c.readRAPLEnergy(raplPath)
		if err == nil && pkgEnergy > 0 {
			pkgPower := c.calculatePowerFromEnergy(fmt.Sprintf("package%d", packageIdx), pkgEnergy)
			if pkgPower > 0 {
				ch <- prometheus.MustNewConstMetric(c.packagePowerDesc, prometheus.GaugeValue, pkgPower, fmt.Sprintf("%d", packageIdx))
				totalPower += pkgPower
			}
		}

		dramPath := fmt.Sprintf("%s/intel-rapl:%d:0", raplPath, packageIdx)
		if _, err := os.Stat(dramPath); err == nil {
			dramEnergy, err := c.readRAPLEnergy(dramPath)
			if err == nil && dramEnergy > 0 {
				dramPower := c.calculatePowerFromEnergy(fmt.Sprintf("dram%d", packageIdx), dramEnergy)
				if dramPower > 0 {
					ch <- prometheus.MustNewConstMetric(c.dramPowerDesc, prometheus.GaugeValue, dramPower, fmt.Sprintf("%d", packageIdx))
					totalPower += dramPower
				}
			}
		}

		packageIdx++
	}

	if totalPower > 0 {
		return totalPower, "rapl", nil
	}

	return 0, "rapl", fmt.Errorf("no RAPL power data")
}

func (c *PowerCollector) readRAPLEnergy(path string) (uint64, error) {
	energyPath := filepath.Join(path, "energy_uj")
	data, err := os.ReadFile(energyPath)
	if err != nil {
		return 0, err
	}

	energy, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}

	return energy, nil
}

func (c *PowerCollector) calculatePowerFromEnergy(key string, currentEnergy uint64) float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	prevEnergy, exists := c.prevEnergyValues[key]
	prevTime := c.prevEnergyTime

	c.prevEnergyValues[key] = currentEnergy
	if c.prevEnergyTime.IsZero() {
		c.prevEnergyTime = now
		return 0
	}
	c.prevEnergyTime = now

	if !exists || prevTime.IsZero() {
		return 0
	}

	timeDelta := now.Sub(prevTime).Seconds()
	if timeDelta <= 0 {
		return 0
	}

	var energyDelta uint64
	if currentEnergy >= prevEnergy {
		energyDelta = currentEnergy - prevEnergy
	} else {
		energyDelta = currentEnergy + (^uint64(0) - prevEnergy)
	}

	power := float64(energyDelta) / 1e6 / timeDelta
	return power
}

func (c *PowerCollector) collectACPI() (float64, string, error) {
	return 0, "acpi", fmt.Errorf("ACPI power monitoring not implemented on this system")
}

func (c *PowerCollector) collectEstimated() (float64, string, error) {
	cpuUsage, err := c.getCPUUsage()
	if err != nil {
		cpuUsage = 0.5
	}

	cpuTDP := c.getCPUTDP()
	cpuPower := cpuTDP * cpuUsage

	gpuPower, err := c.getGPUPower()
	if err != nil {
		gpuPower = 0
	}

	basePower := 50.0

	totalPower := cpuPower + gpuPower + basePower
	return totalPower, "estimated", nil
}

func (c *PowerCollector) getCPUUsage() (float64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) < 1 {
		return 0, fmt.Errorf("invalid /proc/stat")
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 8 || fields[0] != "cpu" {
		return 0, fmt.Errorf("invalid /proc/stat format")
	}

	user, _ := strconv.ParseUint(fields[1], 10, 64)
	nice, _ := strconv.ParseUint(fields[2], 10, 64)
	system, _ := strconv.ParseUint(fields[3], 10, 64)
	idle, _ := strconv.ParseUint(fields[4], 10, 64)
	iowait, _ := strconv.ParseUint(fields[5], 10, 64)
	irq, _ := strconv.ParseUint(fields[6], 10, 64)
	softirq, _ := strconv.ParseUint(fields[7], 10, 64)

	total := user + nice + system + idle + iowait + irq + softirq
	busy := user + nice + system + irq + softirq

	if total == 0 {
		return 0, nil
	}

	return float64(busy) / float64(total), nil
}

func (c *PowerCollector) getCPUTDP() float64 {
	paths := []string{
		"/sys/class/hwmon/hwmon0/power1_max",
		"/sys/class/hwmon/hwmon1/power1_max",
		"/sys/devices/platform/coretemp.0/power1_max",
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		tdpUW, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		if err == nil && tdpUW > 0 {
			return tdpUW / 1e6
		}
	}

	return 150.0
}

func (c *PowerCollector) getGPUPower() (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=power.draw",
		"--format=csv,noheader,nounits")

	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return 0, fmt.Errorf("no GPU power data")
	}

	totalPower := 0.0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "[N/A]" || line == "N/A" {
			continue
		}
		power, err := strconv.ParseFloat(line, 64)
		if err == nil {
			totalPower += power
		}
	}

	return totalPower, nil
}
