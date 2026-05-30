package collector

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/3x-ui-exporter/client"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "xui"

var desc = struct {
	scrapeDuration *prometheus.Desc
	scrapeSuccess  *prometheus.Desc

	clientUpload      *prometheus.Desc
	clientDownload    *prometheus.Desc
	clientTrafficLimit *prometheus.Desc
	clientExpiry      *prometheus.Desc
	clientOnline      *prometheus.Desc
	clientEnabled     *prometheus.Desc

	inboundUp     *prometheus.Desc
	inboundDown   *prometheus.Desc
	inboundTotal  *prometheus.Desc

	serverCPU         *prometheus.Desc
	serverMemCurrent  *prometheus.Desc
	serverMemTotal    *prometheus.Desc
	serverSwapCurrent *prometheus.Desc
	serverSwapTotal   *prometheus.Desc
	serverDiskCurrent *prometheus.Desc
	serverDiskTotal   *prometheus.Desc
	serverNetUp       *prometheus.Desc
	serverNetDown     *prometheus.Desc
	serverXrayRunning *prometheus.Desc
	serverTCPConns    *prometheus.Desc
	serverLoad1       *prometheus.Desc
	serverLoad5       *prometheus.Desc
	serverLoad15      *prometheus.Desc
	serverUptime      *prometheus.Desc

	nodeStatus        *prometheus.Desc
	nodeLatency       *prometheus.Desc
	nodeCPU           *prometheus.Desc
	nodeMem           *prometheus.Desc
	nodeUptime        *prometheus.Desc
	nodeClientCount   *prometheus.Desc
	nodeOnlineCount   *prometheus.Desc
	nodeDepletedCount *prometheus.Desc
	nodeInboundCount  *prometheus.Desc
}{
	scrapeDuration: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "duration_seconds"),
		"Duration of the last scrape in seconds.",
		nil, nil,
	),
	scrapeSuccess: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "success"),
		"Whether the last scrape was successful (1=success, 0=failure).",
		nil, nil,
	),
	clientUpload: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "client", "upload_bytes_total"),
		"Total upload bytes per client.",
		[]string{"email", "inbound_remark", "inbound_id", "protocol", "enable"}, nil,
	),
	clientDownload: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "client", "download_bytes_total"),
		"Total download bytes per client.",
		[]string{"email", "inbound_remark", "inbound_id", "protocol", "enable"}, nil,
	),
	clientTrafficLimit: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "client", "traffic_limit_bytes"),
		"Traffic quota in bytes per client (0 means unlimited).",
		[]string{"email", "inbound_remark", "inbound_id", "protocol", "enable"}, nil,
	),
	clientExpiry: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "client", "expiry_timestamp_seconds"),
		"Client expiry time as unix timestamp (0 means never expires).",
		[]string{"email", "inbound_remark", "inbound_id", "protocol", "enable"}, nil,
	),
	clientOnline: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "client", "online"),
		"Whether the client is currently online (1=online, 0=offline).",
		[]string{"email"}, nil,
	),
	clientEnabled: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "client", "enabled"),
		"Whether the client is enabled (1=enabled, 0=disabled).",
		[]string{"email", "inbound_remark", "inbound_id"}, nil,
	),
	inboundUp: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "inbound", "upload_bytes_total"),
		"Total upload bytes per inbound.",
		[]string{"inbound_remark", "inbound_id", "protocol", "enable"}, nil,
	),
	inboundDown: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "inbound", "download_bytes_total"),
		"Total download bytes per inbound.",
		[]string{"inbound_remark", "inbound_id", "protocol", "enable"}, nil,
	),
	inboundTotal: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "inbound", "traffic_limit_bytes"),
		"Traffic quota per inbound (0 means unlimited).",
		[]string{"inbound_remark", "inbound_id", "protocol", "enable"}, nil,
	),
	serverCPU: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "cpu_usage_percent"),
		"Server CPU usage percentage.",
		nil, nil,
	),
	serverMemCurrent: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "mem_current_bytes"),
		"Server memory currently used.",
		nil, nil,
	),
	serverMemTotal: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "mem_total_bytes"),
		"Server total memory.",
		nil, nil,
	),
	serverSwapCurrent: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "swap_current_bytes"),
		"Server swap currently used.",
		nil, nil,
	),
	serverSwapTotal: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "swap_total_bytes"),
		"Server total swap.",
		nil, nil,
	),
	serverDiskCurrent: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "disk_current_bytes"),
		"Server disk currently used.",
		nil, nil,
	),
	serverDiskTotal: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "disk_total_bytes"),
		"Server total disk.",
		nil, nil,
	),
	serverNetUp: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "net_upload_bytes_total"),
		"Server cumulative network upload bytes.",
		nil, nil,
	),
	serverNetDown: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "net_download_bytes_total"),
		"Server cumulative network download bytes.",
		nil, nil,
	),
	serverXrayRunning: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "xray_running"),
		"Whether Xray is running (1=running, 0=not running).",
		nil, nil,
	),
	serverTCPConns: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "tcp_connections"),
		"Number of TCP connections.",
		nil, nil,
	),
	serverLoad1: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "load1"),
		"1-minute load average.",
		nil, nil,
	),
	serverLoad5: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "load5"),
		"5-minute load average.",
		nil, nil,
	),
	serverLoad15: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "load15"),
		"15-minute load average.",
		nil, nil,
	),
	serverUptime: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "server", "uptime_seconds"),
		"Server uptime in seconds.",
		nil, nil,
	),
	nodeStatus: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "node", "status"),
		"Node status (1=online, 0=offline).",
		[]string{"node_name", "node_id", "address"}, nil,
	),
	nodeLatency: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "node", "latency_ms"),
		"Node latency in milliseconds.",
		[]string{"node_name", "node_id", "address"}, nil,
	),
	nodeCPU: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "node", "cpu_usage_percent"),
		"Node CPU usage percentage.",
		[]string{"node_name", "node_id", "address"}, nil,
	),
	nodeMem: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "node", "mem_usage_percent"),
		"Node memory usage percentage.",
		[]string{"node_name", "node_id", "address"}, nil,
	),
	nodeUptime: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "node", "uptime_seconds"),
		"Node uptime in seconds.",
		[]string{"node_name", "node_id", "address"}, nil,
	),
	nodeClientCount: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "node", "client_count"),
		"Number of clients on the node.",
		[]string{"node_name", "node_id", "address"}, nil,
	),
	nodeOnlineCount: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "node", "online_count"),
		"Number of online clients on the node.",
		[]string{"node_name", "node_id", "address"}, nil,
	),
	nodeDepletedCount: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "node", "depleted_count"),
		"Number of depleted clients on the node.",
		[]string{"node_name", "node_id", "address"}, nil,
	),
	nodeInboundCount: prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "node", "inbound_count"),
		"Number of inbounds on the node.",
		[]string{"node_name", "node_id", "address"}, nil,
	),
}

type Collector struct {
	mu      sync.Mutex
	client  *client.Client
	logger  *slog.Logger
	healthy bool
}

func New(c *client.Client, logger *slog.Logger) *Collector {
	return &Collector{
		client:  c,
		logger:  logger,
		healthy: false,
	}
}

func (col *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- desc.scrapeDuration
	ch <- desc.scrapeSuccess
}

func (col *Collector) Collect(ch chan<- prometheus.Metric) {
	col.mu.Lock()
	defer col.mu.Unlock()

	start := time.Now()
	success := float64(1)

	inbounds, err := col.client.GetInbounds()
	if err != nil {
		col.logger.Error("failed to get inbounds", "error", err)
		success = 0
	}

	onlineClients, err := col.client.GetOnlineClients()
	if err != nil {
		col.logger.Error("failed to get online clients", "error", err)
		success = 0
	}

	serverStatus, err := col.client.GetServerStatus()
	if err != nil {
		col.logger.Error("failed to get server status", "error", err)
		success = 0
	}

	nodes, err := col.client.GetNodes()
	if err != nil {
		col.logger.Warn("failed to get nodes (non-fatal)", "error", err)
	}

	col.healthy = success == 1

	onlineSet := make(map[string]bool)
	for _, email := range onlineClients {
		onlineSet[email] = true
	}

	if inbounds != nil {
		collectInboundMetrics(ch, inbounds, onlineSet)
	}

	if serverStatus != nil {
		collectServerMetrics(ch, serverStatus)
	}

	if nodes != nil {
		collectNodeMetrics(ch, nodes)
	}

	duration := time.Since(start).Seconds()
	ch <- prometheus.MustNewConstMetric(desc.scrapeDuration, prometheus.GaugeValue, duration)
	ch <- prometheus.MustNewConstMetric(desc.scrapeSuccess, prometheus.GaugeValue, success)
}

func (col *Collector) IsHealthy() bool {
	col.mu.Lock()
	defer col.mu.Unlock()
	return col.healthy
}

func collectInboundMetrics(ch chan<- prometheus.Metric, inbounds []client.Inbound, onlineSet map[string]bool) {
	allEmails := make(map[string]bool)

	for _, ib := range inbounds {
		remark := sanitizeLabel(ib.Remark)
		ibID := fmt.Sprintf("%d", ib.ID)
		enableStr := fmt.Sprintf("%t", ib.Enable)

		ch <- prometheus.MustNewConstMetric(desc.inboundUp, prometheus.GaugeValue, float64(ib.Up), remark, ibID, ib.Protocol, enableStr)
		ch <- prometheus.MustNewConstMetric(desc.inboundDown, prometheus.GaugeValue, float64(ib.Down), remark, ibID, ib.Protocol, enableStr)
		ch <- prometheus.MustNewConstMetric(desc.inboundTotal, prometheus.GaugeValue, float64(ib.Total), remark, ibID, ib.Protocol, enableStr)

		for _, cs := range ib.ClientStats {
			email := sanitizeLabel(cs.Email)
			csEnableStr := fmt.Sprintf("%t", cs.Enable)

			if email != "" {
				allEmails[email] = true
			}

			ch <- prometheus.MustNewConstMetric(desc.clientUpload, prometheus.GaugeValue, float64(cs.Up), email, remark, ibID, ib.Protocol, csEnableStr)
			ch <- prometheus.MustNewConstMetric(desc.clientDownload, prometheus.GaugeValue, float64(cs.Down), email, remark, ibID, ib.Protocol, csEnableStr)
			ch <- prometheus.MustNewConstMetric(desc.clientTrafficLimit, prometheus.GaugeValue, float64(cs.Total), email, remark, ibID, ib.Protocol, csEnableStr)

			var expirySeconds float64
			if cs.ExpiryTime > 0 {
				expirySeconds = float64(cs.ExpiryTime) / 1000.0
			}
			ch <- prometheus.MustNewConstMetric(desc.clientExpiry, prometheus.GaugeValue, expirySeconds, email, remark, ibID, ib.Protocol, csEnableStr)

			ch <- prometheus.MustNewConstMetric(desc.clientEnabled, prometheus.GaugeValue, boolToF(cs.Enable), email, remark, ibID)
		}
	}

	for email := range allEmails {
		val := 0.0
		if onlineSet[email] {
			val = 1.0
		}
		ch <- prometheus.MustNewConstMetric(desc.clientOnline, prometheus.GaugeValue, val, email)
	}
}

func collectServerMetrics(ch chan<- prometheus.Metric, s *client.ServerStatus) {
	ch <- prometheus.MustNewConstMetric(desc.serverCPU, prometheus.GaugeValue, s.CPU)
	ch <- prometheus.MustNewConstMetric(desc.serverMemCurrent, prometheus.GaugeValue, float64(s.Mem.Current))
	ch <- prometheus.MustNewConstMetric(desc.serverMemTotal, prometheus.GaugeValue, float64(s.Mem.Total))
	ch <- prometheus.MustNewConstMetric(desc.serverSwapCurrent, prometheus.GaugeValue, float64(s.Swap.Current))
	ch <- prometheus.MustNewConstMetric(desc.serverSwapTotal, prometheus.GaugeValue, float64(s.Swap.Total))
	ch <- prometheus.MustNewConstMetric(desc.serverDiskCurrent, prometheus.GaugeValue, float64(s.Disk.Current))
	ch <- prometheus.MustNewConstMetric(desc.serverDiskTotal, prometheus.GaugeValue, float64(s.Disk.Total))
	ch <- prometheus.MustNewConstMetric(desc.serverNetUp, prometheus.GaugeValue, float64(s.NetIO.Up))
	ch <- prometheus.MustNewConstMetric(desc.serverNetDown, prometheus.GaugeValue, float64(s.NetIO.Down))

	xrayRunning := 0.0
	if s.Xray.State == "running" {
		xrayRunning = 1.0
	}
	ch <- prometheus.MustNewConstMetric(desc.serverXrayRunning, prometheus.GaugeValue, xrayRunning)
	ch <- prometheus.MustNewConstMetric(desc.serverTCPConns, prometheus.GaugeValue, float64(s.TCPCount))
	ch <- prometheus.MustNewConstMetric(desc.serverLoad1, prometheus.GaugeValue, s.Load.Load1)
	ch <- prometheus.MustNewConstMetric(desc.serverLoad5, prometheus.GaugeValue, s.Load.Load5)
	ch <- prometheus.MustNewConstMetric(desc.serverLoad15, prometheus.GaugeValue, s.Load.Load15)
	ch <- prometheus.MustNewConstMetric(desc.serverUptime, prometheus.GaugeValue, float64(s.Uptime))
}

func collectNodeMetrics(ch chan<- prometheus.Metric, nodes []client.Node) {
	for _, n := range nodes {
		name := sanitizeLabel(n.Name)
		nodeID := fmt.Sprintf("%d", n.ID)
		addr := fmt.Sprintf("%s:%d", n.Address, n.Port)

		online := 0.0
		if n.Status == "online" {
			online = 1.0
		}
		ch <- prometheus.MustNewConstMetric(desc.nodeStatus, prometheus.GaugeValue, online, name, nodeID, addr)
		ch <- prometheus.MustNewConstMetric(desc.nodeLatency, prometheus.GaugeValue, float64(n.LatencyMs), name, nodeID, addr)
		ch <- prometheus.MustNewConstMetric(desc.nodeCPU, prometheus.GaugeValue, n.CPUPct, name, nodeID, addr)
		ch <- prometheus.MustNewConstMetric(desc.nodeMem, prometheus.GaugeValue, n.MemPct, name, nodeID, addr)
		ch <- prometheus.MustNewConstMetric(desc.nodeUptime, prometheus.GaugeValue, float64(n.UptimeSecs), name, nodeID, addr)
		ch <- prometheus.MustNewConstMetric(desc.nodeClientCount, prometheus.GaugeValue, float64(n.ClientCount), name, nodeID, addr)
		ch <- prometheus.MustNewConstMetric(desc.nodeOnlineCount, prometheus.GaugeValue, float64(n.OnlineCount), name, nodeID, addr)
		ch <- prometheus.MustNewConstMetric(desc.nodeDepletedCount, prometheus.GaugeValue, float64(n.DepletedCount), name, nodeID, addr)
		ch <- prometheus.MustNewConstMetric(desc.nodeInboundCount, prometheus.GaugeValue, float64(n.InboundCount), name, nodeID, addr)
	}
}

func sanitizeLabel(s string) string {
	return strings.TrimSpace(s)
}

func boolToF(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
