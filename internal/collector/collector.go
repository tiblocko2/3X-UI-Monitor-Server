// Package collector периодически собирает системные метрики и пишет их в Store.
package collector

import (
	"log"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"

	"vpn-monitor/internal/config"
	"vpn-monitor/internal/store"
	"vpn-monitor/internal/xui"
)

// Collector собирает метрики и записывает DataPoint-ы в store.
type Collector struct {
	cfg       config.Collector
	store     *store.Store
	xuiClient *xui.Client

	// состояние для расчёта скорости сети
	prevNetIO   []psnet.IOCountersStat
	prevNetTime time.Time
}

// New создаёт Collector.
func New(cfg config.Collector, s *store.Store, xuiClient *xui.Client) *Collector {
	return &Collector{
		cfg:       cfg,
		store:     s,
		xuiClient: xuiClient,
	}
}

// Run запускает бесконечный цикл сбора метрик. Вызывать в отдельной горутине.
func (c *Collector) Run() {
	// Первый сбор сразу, без ожидания первого тика.
	c.collect()

	ticker := time.NewTicker(c.cfg.Interval)
	defer ticker.Stop()

	for range ticker.C {
		c.collect()
	}
}

func (c *Collector) collect() {
	dp := store.DataPoint{
		Timestamp: time.Now().UnixMilli(),
		CPU:       c.cpuPercent(),
		RAM:       c.ramPercent(),
		Clients:   c.xuiClient.OnlineCount(),
	}
	dp.NetIn, dp.NetOut = c.netSpeed()

	// -1 означает ошибку связи с 3X-UI; на графике показываем 0.
	if dp.Clients < 0 {
		dp.Clients = 0
	}

	c.store.Add(dp)
}

func (c *Collector) cpuPercent() float64 {
	pct, err := cpu.Percent(time.Second, false)
	if err != nil || len(pct) == 0 {
		log.Printf("[collector] cpu: %v", err)
		return 0
	}
	return pct[0]
}

func (c *Collector) ramPercent() float64 {
	vm, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("[collector] ram: %v", err)
		return 0
	}
	return vm.UsedPercent
}

// netSpeed возвращает скорость входящего и исходящего трафика в Mbps.
func (c *Collector) netSpeed() (inMbps, outMbps float64) {
	now := time.Now()

	counters, err := psnet.IOCounters(false)
	if err != nil || len(counters) == 0 {
		log.Printf("[collector] net: %v", err)
		return 0, 0
	}
	current := counters[0]

	// Первый вызов — только сохраняем baseline, скорость ещё посчитать нельзя.
	if c.prevNetTime.IsZero() {
		c.prevNetIO = counters
		c.prevNetTime = now
		return 0, 0
	}

	dt := now.Sub(c.prevNetTime).Seconds()
	if dt <= 0 {
		return 0, 0
	}

	inBytes := float64(current.BytesRecv - c.prevNetIO[0].BytesRecv)
	outBytes := float64(current.BytesSent - c.prevNetIO[0].BytesSent)

	// bytes → Mbps: умножаем на 8 бит, делим на 1 000 000
	inMbps = (inBytes * 8) / 1_000_000 / dt
	outMbps = (outBytes * 8) / 1_000_000 / dt

	c.prevNetIO = counters
	c.prevNetTime = now
	return inMbps, outMbps
}
