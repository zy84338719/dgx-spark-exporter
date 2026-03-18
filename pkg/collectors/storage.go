package collectors

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/zy84338719/dgx-spark-exporter/internal/config"
	"github.com/zy84338719/dgx-spark-exporter/pkg/collectors/utils"

	"github.com/prometheus/client_golang/prometheus"
)

type StorageCollector struct {
	nvmeTempDesc *prometheus.Desc
	diskTempDesc *prometheus.Desc

	logger *slog.Logger
}

func NewStorageCollector(cfg *config.Config, logger *slog.Logger) *StorageCollector {
	return &StorageCollector{
		nvmeTempDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "nvme", "temperature_celsius"),
			"NVMe device temperature in degrees Celsius",
			[]string{"device"}, nil,
		),
		diskTempDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "disk", "temperature_celsius"),
			"Disk device temperature in degrees Celsius",
			[]string{"device"}, nil,
		),
		logger: logger.With("collector", "storage"),
	}
}

func (c *StorageCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.nvmeTempDesc
	ch <- c.diskTempDesc
}

func (c *StorageCollector) Collect(ch chan<- prometheus.Metric) {
	c.collectHwmon(ch)
}

func (c *StorageCollector) collectHwmon(ch chan<- prometheus.Metric) {
	hwmonPath := "/sys/class/hwmon"
	entries, err := os.ReadDir(hwmonPath)
	if err != nil {
		c.logger.Debug("failed to read hwmon directory", "error", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			continue
		}

		hwmonDir := filepath.Join(hwmonPath, entry.Name())
		nameData, err := os.ReadFile(filepath.Join(hwmonDir, "name"))
		if err != nil {
			continue
		}

		deviceName := strings.TrimSpace(string(nameData))
		deviceNameLower := strings.ToLower(deviceName)

		if strings.Contains(deviceNameLower, "nvme") {
			c.collectNVMeTemp(ch, hwmonDir, deviceName)
		} else if strings.Contains(deviceNameLower, "sd") ||
			strings.Contains(deviceNameLower, "hd") ||
			strings.Contains(deviceNameLower, "disk") {
			c.collectDiskTemp(ch, hwmonDir, deviceName)
		}
	}
}

func (c *StorageCollector) collectNVMeTemp(ch chan<- prometheus.Metric, hwmonDir, deviceName string) {
	files, err := filepath.Glob(filepath.Join(hwmonDir, "temp*_input"))
	if err != nil {
		return
	}

	for _, tempFile := range files {
		temp, err := utils.ReadFileFloat(tempFile)
		if err != nil {
			continue
		}

		tempCelsius := temp / 1000.0

		deviceLabel := deviceName
		labelFile := strings.Replace(tempFile, "_input", "_label", 1)
		if labelData, err := os.ReadFile(labelFile); err == nil {
			label := strings.TrimSpace(string(labelData))
			if label != "" {
				deviceLabel = deviceName + "_" + label
			}
		}

		ch <- prometheus.MustNewConstMetric(c.nvmeTempDesc, prometheus.GaugeValue, tempCelsius, deviceLabel)
	}
}

func (c *StorageCollector) collectDiskTemp(ch chan<- prometheus.Metric, hwmonDir, deviceName string) {
	files, err := filepath.Glob(filepath.Join(hwmonDir, "temp*_input"))
	if err != nil {
		return
	}

	for _, tempFile := range files {
		temp, err := utils.ReadFileFloat(tempFile)
		if err != nil {
			continue
		}

		tempCelsius := temp / 1000.0
		ch <- prometheus.MustNewConstMetric(c.diskTempDesc, prometheus.GaugeValue, tempCelsius, deviceName)
	}
}
