package collectors

import (
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zy84338719/dgx-spark-exporter/internal/config"

	"github.com/prometheus/client_golang/prometheus"
)

type NetworkCollector struct {
	rxBytesDesc   *prometheus.Desc
	txBytesDesc   *prometheus.Desc
	rxPacketsDesc *prometheus.Desc
	txPacketsDesc *prometheus.Desc
	rxErrorsDesc  *prometheus.Desc
	txErrorsDesc  *prometheus.Desc
	rxDroppedDesc *prometheus.Desc
	txDroppedDesc *prometheus.Desc

	interfaces []string
	logger     *slog.Logger
}

func NewNetworkCollector(cfg *config.Config, logger *slog.Logger) *NetworkCollector {
	return &NetworkCollector{
		rxBytesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "network", "receive_bytes_total"),
			"Total bytes received on network interface",
			[]string{"interface"}, nil,
		),
		txBytesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "network", "transmit_bytes_total"),
			"Total bytes transmitted on network interface",
			[]string{"interface"}, nil,
		),
		rxPacketsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "network", "receive_packets_total"),
			"Total packets received on network interface",
			[]string{"interface"}, nil,
		),
		txPacketsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "network", "transmit_packets_total"),
			"Total packets transmitted on network interface",
			[]string{"interface"}, nil,
		),
		rxErrorsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "network", "receive_errors_total"),
			"Total receive errors on network interface",
			[]string{"interface"}, nil,
		),
		txErrorsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "network", "transmit_errors_total"),
			"Total transmit errors on network interface",
			[]string{"interface"}, nil,
		),
		rxDroppedDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "network", "receive_dropped_total"),
			"Total receive dropped on network interface",
			[]string{"interface"}, nil,
		),
		txDroppedDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "network", "transmit_dropped_total"),
			"Total transmit dropped on network interface",
			[]string{"interface"}, nil,
		),
		interfaces: cfg.NetworkInterfaces,
		logger:     logger.With("collector", "network"),
	}
}

func (c *NetworkCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.rxBytesDesc
	ch <- c.txBytesDesc
	ch <- c.rxPacketsDesc
	ch <- c.txPacketsDesc
	ch <- c.rxErrorsDesc
	ch <- c.txErrorsDesc
	ch <- c.rxDroppedDesc
	ch <- c.txDroppedDesc
}

func (c *NetworkCollector) Collect(ch chan<- prometheus.Metric) {
	for _, iface := range c.interfaces {
		if !isInterfaceUp(iface) {
			continue
		}

		statsDir := filepath.Join("/sys/class/net", iface, "statistics")

		rxBytes := readSysUint64(filepath.Join(statsDir, "rx_bytes"))
		txBytes := readSysUint64(filepath.Join(statsDir, "tx_bytes"))
		rxPackets := readSysUint64(filepath.Join(statsDir, "rx_packets"))
		txPackets := readSysUint64(filepath.Join(statsDir, "tx_packets"))
		rxErrors := readSysUint64(filepath.Join(statsDir, "rx_errors"))
		txErrors := readSysUint64(filepath.Join(statsDir, "tx_errors"))
		rxDropped := readSysUint64(filepath.Join(statsDir, "rx_dropped"))
		txDropped := readSysUint64(filepath.Join(statsDir, "tx_dropped"))

		ch <- prometheus.MustNewConstMetric(c.rxBytesDesc, prometheus.CounterValue, float64(rxBytes), iface)
		ch <- prometheus.MustNewConstMetric(c.txBytesDesc, prometheus.CounterValue, float64(txBytes), iface)
		ch <- prometheus.MustNewConstMetric(c.rxPacketsDesc, prometheus.CounterValue, float64(rxPackets), iface)
		ch <- prometheus.MustNewConstMetric(c.txPacketsDesc, prometheus.CounterValue, float64(txPackets), iface)
		ch <- prometheus.MustNewConstMetric(c.rxErrorsDesc, prometheus.CounterValue, float64(rxErrors), iface)
		ch <- prometheus.MustNewConstMetric(c.txErrorsDesc, prometheus.CounterValue, float64(txErrors), iface)
		ch <- prometheus.MustNewConstMetric(c.rxDroppedDesc, prometheus.CounterValue, float64(rxDropped), iface)
		ch <- prometheus.MustNewConstMetric(c.txDroppedDesc, prometheus.CounterValue, float64(txDropped), iface)
	}
}

func isInterfaceUp(iface string) bool {
	path := filepath.Join("/sys/class/net", iface, "operstate")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "up"
}

func readSysUint64(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return v
}
