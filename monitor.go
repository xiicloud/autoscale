package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

const scaleDelay = 5

type Stat struct {
	CPUStats struct {
		CPUUsage struct {
			PercpuUsage       []float64 `json:"percpu_usage"`
			TotalUsage        float64   `json:"total_usage"`
			UsageInKernelmode float64   `json:"usage_in_kernelmode"`
			UsageInUsermode   float64   `json:"usage_in_usermode"`
		} `json:"cpu_usage"`
		SystemCPUUsage float64 `json:"system_cpu_usage"`
	} `json:"cpu_stats"`
	MemoryStats struct {
		Failcnt  float64 `json:"failcnt"`
		Limit    float64 `json:"limit"`
		MaxUsage float64 `json:"max_usage"`
		Usage    float64 `json:"usage"`
	} `json:"memory_stats"`
}

type watcher struct {
	cid      string
	lastStat *Stat
	stop     chan bool
	m        *monitor
}

func newWatcher(cid string, m *monitor) *watcher {
	return &watcher{
		cid:  cid,
		stop: make(chan bool),
		m:    m,
	}
}

func (w *watcher) quit() {
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
}

func (w *watcher) watch() (err error) {
	url := controllerAddr + "/api/containers/" + w.cid + "/stats?ApiKey=" + apiKey
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		res.Body.Close()
		w.m.evict(w.cid)
		logrus.Errorf("watch error: %v", err)
	}()

	r := bufio.NewReader(res.Body)
	for {
		select {
		case <-w.stop:
			return nil
		default:
		}

		line, err := r.ReadString('\n')
		if err != nil {
			return err
		}

		if len(line) < 100 {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		line = strings.TrimSpace(parts[1])
		stat := &Stat{}
		err = json.Unmarshal([]byte(line), stat)
		if err != nil {
			return err
		}

		if w.lastStat == nil {
			w.lastStat = stat
			continue
		}

		cpu := (stat.CPUStats.CPUUsage.TotalUsage - w.lastStat.CPUStats.CPUUsage.TotalUsage) * 100 / (stat.CPUStats.SystemCPUUsage - w.lastStat.CPUStats.SystemCPUUsage)
		memory := stat.MemoryStats.Usage
		w.m.setMetrics(w.cid, cpu, memory)

		w.lastStat = stat
	}
}

type monitor struct {
	*AutoScaleGroup
	sync.Mutex
	watchers map[string]*watcher
	cpu      map[string]float64
	mem      map[string]float64
	recent5  []int8 // 0: no scale events. 1: scale out. -1: scale in.
}

func newMonitor(asg *AutoScaleGroup) *monitor {
	return &monitor{
		AutoScaleGroup: asg,
		watchers:       make(map[string]*watcher),
		cpu:            make(map[string]float64),
		mem:            make(map[string]float64),
		recent5:        make([]int8, 0, 5),
	}
}

func (m *monitor) watchContainersChange() {
	for range time.Tick(time.Second) {
		containers, err := listContainers(m.App, m.Service)
		if err != nil {
			logrus.Errorf("Failed to call container list API: %v", err)
			continue
		}

		m.Lock()
		cids := make(map[string]bool)
		for _, c := range containers {
			cids[c.Id] = true
			if _, ok := m.watchers[c.Id]; !ok {
				// start watcher for the container
				m.watchers[c.Id] = newWatcher(c.Id, m)
				go m.watchers[c.Id].watch()
			}
		}

		// cleanup stale watchers
		for id := range m.watchers {
			if !cids[id] {
				m.evictUnsafe(id)
			}
		}
		m.Unlock()
	}
}

func sum(vars []int8) int8 {
	var s int8
	for _, v := range vars {
		s += v
	}
	return s
}

func (m *monitor) start() {
	go m.watchContainersChange()

	for range time.Tick(time.Second) {
		m.Lock()
		logrus.Debugf("monitors count: %d", len(m.watchers))

		avgMem := avg(m.mem)
		avgCpu := avg(m.cpu)
		if len(m.recent5) == 5 {
			m.recent5 = m.recent5[1:]
		}

		if avgCpu >= m.CpuHigh || avgMem >= m.MemoryHigh {
			m.recent5 = append(m.recent5, 1)
		} else if avgCpu <= m.CpuLow && avgMem <= m.MemoryLow {
			m.recent5 = append(m.recent5, -1)
		} else {
			m.recent5 = append(m.recent5, 0)
		}

		scaleOut := false
		scaleIn := false
		x := sum(m.recent5)
		switch x {
		case 5:
			scaleOut = true
		case -5:
			scaleIn = true
		default:
			m.Unlock()
			logrus.Debugf("sum: %d, cpu:%f, mem:%f, no need to scale", x, avgCpu, avgMem)
			continue
		}
		m.recent5 = make([]int8, 0, 5)

		currentContainers := float64(len(m.watchers))
		if scaleIn && currentContainers <= m.MinContainers {
			logrus.Debugf("containers limit(less than %d) reached", int(m.MinContainers))
			m.Unlock()
			continue
		}

		if scaleOut && currentContainers >= m.MaxContainers {
			logrus.Debugf("containers limit(more than %d) reached", int(m.MaxContainers))
			m.Unlock()
			continue
		}

		m.Unlock()

		if scaleOut {
			if err := addContainer(m.App, m.Service); err != nil {
				logrus.Errorf("Failed to scale out %s.%s: %v", m.App, m.Service, err)
			} else {
				logrus.Infof("Added 1 new container to %s.%s", m.App, m.Service)
			}
		} else if scaleIn {
			if err := delContainer(m.App, m.Service); err != nil {
				logrus.Errorf("Failed to scale in %s.%s: %v", m.App, m.Service, err)
			} else {
				logrus.Infof("Deleted 1 new container from %s.%s", m.App, m.Service)
			}
		}
	}
}

func (m *monitor) evictUnsafe(cid string) {
	if _, ok := m.mem[cid]; ok {
		delete(m.mem, cid)
	}
	if _, ok := m.cpu[cid]; ok {
		delete(m.cpu, cid)
	}
	if _, ok := m.watchers[cid]; ok {
		m.watchers[cid].quit()
		delete(m.watchers, cid)
	}
}

func (m *monitor) evict(cid string) {
	m.Lock()
	m.evictUnsafe(cid)
	m.Unlock()
}

func (m *monitor) setMetrics(cid string, cpu, mem float64) {
	m.Lock()
	m.cpu[cid] = cpu
	m.mem[cid] = mem
	m.Unlock()
}

func avg(m map[string]float64) float64 {
	var r float64
	for _, v := range m {
		r += v
	}
	return r / float64(len(m))
}
