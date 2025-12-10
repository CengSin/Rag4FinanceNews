package handle

import (
	"fmt"
	"github.com/go-mysql-org/go-mysql/canal"
	"log"
	"rag4financenew/config"
	"time"
)

func ChangDataCapture(cfg *config.Cdc) {
	// 1. 配置 Canal
	canalCfg := canal.NewDefaultConfig()
	canalCfg.Addr = cfg.Addr
	canalCfg.User = cfg.User
	canalCfg.Password = cfg.Password // 替换你的密码
	//canalCfg.Dump.TableDB = cfg.DbName
	//canalCfg.Dump.Tables = cfg.TableName
	canalCfg.Dump.ExecutionPath = "" // 如果不需要全量 dump，可以留空或指向 mysqldump
	var tableRegex []string
	for _, tbName := range cfg.TableName {
		tableRegex = append(tableRegex, fmt.Sprintf("%s\\.%s", cfg.DbName, tbName))
	}
	canalCfg.IncludeTableRegex = tableRegex
	canalCfg.ReadTimeout = time.Second * 30
	canalCfg.HeartbeatPeriod = time.Second * 30

	// 2. 创建 Canal 实例
	c, err := canal.NewCanal(canalCfg)
	if err != nil {
		log.Fatal(err)
	}

	// 3. 注册我们的处理器
	c.SetEventHandler(&NewsHandler{})

	// 4. 开始运行
	// StartFrom() 会尝试从最新的位置开始监听
	// 实际上生产环境我们需要从上次保存的位置（Position）开始，这里先简化
	log.Println("开始监听 MySQL Binlog...")
	if err = c.Run(); err != nil {
		log.Fatal(err)
	}
}
