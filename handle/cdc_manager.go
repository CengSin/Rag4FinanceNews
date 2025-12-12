package handle

import (
	"fmt"
	"github.com/go-mysql-org/go-mysql/canal"
	"log"
	"rag4financenew/config"
	"strings"
	"sync"
	"time"
)

type cdcMonitor struct {
	cfg      config.Cdc
	canal    *canal.Canal
	stopOnce sync.Once
}

var (
	cdcMu       sync.Mutex
	cdcMonitors = make(map[string]*cdcMonitor)
	defaultCDC  []config.Cdc
)

// SetDefaultCDC 保存启动时读取的配置，在未指定配置时用于 API 默认值
func SetDefaultCDC(cfgs []config.Cdc) {
	cdcMu.Lock()
	defer cdcMu.Unlock()
	defaultCDC = cfgs
}

// GetDefaultCDC 返回默认 CDC 配置的副本
func GetDefaultCDC() []config.Cdc {
	cdcMu.Lock()
	defer cdcMu.Unlock()
	dst := make([]config.Cdc, len(defaultCDC))
	copy(dst, defaultCDC)
	return dst
}

// StartCDC 启动传入配置的监听，返回成功启动的 key 列表
func StartCDC(cfgs []config.Cdc) ([]string, error) {
	started := make([]string, 0, len(cfgs))
	for _, cfg := range cfgs {
		key := cdcKey(cfg)

		cdcMu.Lock()
		_, exists := cdcMonitors[key]
		cdcMu.Unlock()
		if exists {
			continue
		}

		c, err := newCanal(cfg)
		if err != nil {
			return started, err
		}

		monitor := &cdcMonitor{cfg: cfg, canal: c}
		cdcMu.Lock()
		cdcMonitors[key] = monitor
		cdcMu.Unlock()

		go func(k string, m *cdcMonitor) {
			log.Printf("CDC 开始监听 %s\n", k)
			defer func() {
				cdcMu.Lock()
				delete(cdcMonitors, k)
				cdcMu.Unlock()
			}()
			if err := m.canal.Run(); err != nil {
				log.Printf("CDC 监听 %s 退出，err: %v\n", k, err)
			}
		}(key, monitor)

		started = append(started, key)
	}
	return started, nil
}

// StopAllCDC 关闭所有 CDC 监听，返回已关闭的 key 列表
func StopAllCDC() []string {
	cdcMu.Lock()
	defer cdcMu.Unlock()

	keys := make([]string, 0, len(cdcMonitors))
	for key, monitor := range cdcMonitors {
		monitor.stopOnce.Do(func() {
			monitor.canal.Close()
		})
		keys = append(keys, key)
	}

	// 清理 map
	cdcMonitors = make(map[string]*cdcMonitor)
	return keys
}

func cdcKey(cfg config.Cdc) string {
	return fmt.Sprintf("%s/%s/%s", cfg.Addr, cfg.DbName, strings.Join(cfg.TableName, ","))
}

func newCanal(cfg config.Cdc) (*canal.Canal, error) {
	canalCfg := canal.NewDefaultConfig()
	canalCfg.Addr = cfg.Addr
	canalCfg.User = cfg.User
	canalCfg.Password = cfg.Password
	canalCfg.Dump.ExecutionPath = "" // 仅增量

	var tableRegex []string
	for _, tbName := range cfg.TableName {
		tableRegex = append(tableRegex, fmt.Sprintf("%s\\.%s", cfg.DbName, tbName))
	}
	canalCfg.IncludeTableRegex = tableRegex
	canalCfg.ReadTimeout = time.Second * 30
	canalCfg.HeartbeatPeriod = time.Second * 30

	c, err := canal.NewCanal(canalCfg)
	if err != nil {
		return nil, err
	}

	c.SetEventHandler(&NewsHandler{})
	return c, nil
}
