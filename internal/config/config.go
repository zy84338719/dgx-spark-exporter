package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	ListenAddr        string
	MetricsPath       string
	NetworkInterfaces []string
	ScrapeTimeout     time.Duration
	LogLevel          string
	Collectors        []string
	RootMountPoint    string
	ThermalZoneCount  int
	MaxCPUCount       int
	PowerSource       string
	PowercapPath      string
}

func Load() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.ListenAddr, "listen", getEnv("LISTEN_ADDR", ":9876"), "Address to listen on")
	flag.StringVar(&cfg.MetricsPath, "metrics-path", getEnv("METRICS_PATH", "/metrics"), "Path for Prometheus metrics")
	flag.StringVar(&cfg.LogLevel, "log-level", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
	flag.DurationVar(&cfg.ScrapeTimeout, "scrape-timeout", 10*time.Second, "Scrape timeout duration")
	flag.StringVar(&cfg.RootMountPoint, "root-mount", getEnv("ROOT_MOUNT", "/"), "Root mount point for storage metrics")
	flag.IntVar(&cfg.ThermalZoneCount, "thermal-zones", 10, "Number of thermal zones to scan")
	flag.IntVar(&cfg.MaxCPUCount, "max-cpus", 256, "Maximum number of CPU cores to scan")
	flag.StringVar(&cfg.PowerSource, "power-source", getEnv("POWER_SOURCE", "auto"), "Power monitoring source (auto, rapl, acpi, estimated)")
	flag.StringVar(&cfg.PowercapPath, "powercap-path", getEnv("POWERCAP_PATH", "/sys/class/powercap"), "Path to powercap interface")

	ifaces := flag.String("interfaces", getEnv("NETWORK_INTERFACES", "enP7s7,enp1s0f1np1,enP2p1s0f1np1,enp1s0f0np0,enP2p1s0f0np0,wlP9s9"), "Comma-separated list of network interfaces to monitor")

	collectors := flag.String("collectors", getEnv("COLLECTORS", "cpu,gpu,memory,disk,network,system,swap,power,storage"), "Comma-separated list of collectors to enable")

	flag.Parse()

	cfg.NetworkInterfaces = parseStringList(*ifaces)
	cfg.Collectors = parseStringList(*collectors)

	return cfg
}

func (c *Config) IsCollectorEnabled(name string) bool {
	for _, col := range c.Collectors {
		if col == name || col == "all" {
			return true
		}
	}
	return false
}

func parseStringList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{ListenAddr: %s, MetricsPath: %s, LogLevel: %s, Collectors: %v, Interfaces: %v}",
		c.ListenAddr, c.MetricsPath, c.LogLevel, c.Collectors, c.NetworkInterfaces)
}
