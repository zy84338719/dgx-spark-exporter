package collectors

import (
	"bufio"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/zy84338719/dgx-spark-exporter/internal/config"

	"github.com/prometheus/client_golang/prometheus"
)

type DiskCollector struct {
	readsDesc      *prometheus.Desc
	writesDesc     *prometheus.Desc
	readBytesDesc  *prometheus.Desc
	writeBytesDesc *prometheus.Desc
	usedRatioDesc  *prometheus.Desc
	totalBytesDesc *prometheus.Desc
	availBytesDesc *prometheus.Desc

	rootMount string
	logger    *slog.Logger
}

var (
	physicalPrefixes = []string{"sd", "nvme", "vd", "hd", "xvd", "mmcblk"}
	excludePrefixes  = []string{"loop", "ram", "dm-", "sr", "fd"}
)

func NewDiskCollector(cfg *config.Config, logger *slog.Logger) *DiskCollector {
	return &DiskCollector{
		readsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "disk", "reads_completed_total"),
			"Total number of completed disk read operations",
			[]string{"device"}, nil,
		),
		writesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "disk", "writes_completed_total"),
			"Total number of completed disk write operations",
			[]string{"device"}, nil,
		),
		readBytesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "disk", "read_bytes_total"),
			"Total number of bytes read from disk",
			[]string{"device"}, nil,
		),
		writeBytesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "disk", "written_bytes_total"),
			"Total number of bytes written to disk",
			[]string{"device"}, nil,
		),
		usedRatioDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "filesystem", "used_ratio"),
			"Used storage ratio of filesystem (0-1)",
			[]string{"mountpoint"}, nil,
		),
		totalBytesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "filesystem", "size_bytes"),
			"Total size of filesystem in bytes",
			[]string{"mountpoint"}, nil,
		),
		availBytesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "filesystem", "avail_bytes"),
			"Available space on filesystem in bytes",
			[]string{"mountpoint"}, nil,
		),
		rootMount: cfg.RootMountPoint,
		logger:    logger.With("collector", "disk"),
	}
}

func (c *DiskCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.readsDesc
	ch <- c.writesDesc
	ch <- c.readBytesDesc
	ch <- c.writeBytesDesc
	ch <- c.usedRatioDesc
	ch <- c.totalBytesDesc
	ch <- c.availBytesDesc
}

func (c *DiskCollector) Collect(ch chan<- prometheus.Metric) {
	c.collectDiskIO(ch)
	c.collectFilesystem(ch)
}

func (c *DiskCollector) collectDiskIO(ch chan<- prometheus.Metric) {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		c.logger.Debug("failed to open /proc/diskstats", "error", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 {
			continue
		}

		device := fields[2]

		if hasPrefix(device, excludePrefixes) {
			continue
		}

		if !hasPrefix(device, physicalPrefixes) {
			continue
		}

		reads, _ := strconv.ParseFloat(fields[3], 64)
		writes, _ := strconv.ParseFloat(fields[7], 64)
		readBytes, _ := strconv.ParseFloat(fields[9], 64)
		writeBytes, _ := strconv.ParseFloat(fields[13], 64)

		ch <- prometheus.MustNewConstMetric(c.readsDesc, prometheus.CounterValue, reads, device)
		ch <- prometheus.MustNewConstMetric(c.writesDesc, prometheus.CounterValue, writes, device)
		ch <- prometheus.MustNewConstMetric(c.readBytesDesc, prometheus.CounterValue, readBytes*512, device)
		ch <- prometheus.MustNewConstMetric(c.writeBytesDesc, prometheus.CounterValue, writeBytes*512, device)
	}
}

func (c *DiskCollector) collectFilesystem(ch chan<- prometheus.Metric) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(c.rootMount, &stat); err != nil {
		c.logger.Debug("failed to stat filesystem", "path", c.rootMount, "error", err)
		return
	}

	total := stat.Blocks * uint64(stat.Bsize)
	avail := stat.Bavail * uint64(stat.Bsize)

	if total == 0 {
		return
	}

	usedRatio := float64(total-avail) / float64(total)

	ch <- prometheus.MustNewConstMetric(c.usedRatioDesc, prometheus.GaugeValue, usedRatio, c.rootMount)
	ch <- prometheus.MustNewConstMetric(c.totalBytesDesc, prometheus.GaugeValue, float64(total), c.rootMount)
	ch <- prometheus.MustNewConstMetric(c.availBytesDesc, prometheus.GaugeValue, float64(avail), c.rootMount)
}

func hasPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
