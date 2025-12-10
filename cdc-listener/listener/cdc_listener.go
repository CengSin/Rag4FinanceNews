package main

import (
	"context"
	"fmt"
	"github.com/go-mysql-org/go-mysql/canal"
	"go.temporal.io/sdk/client"
	"log"
	cdc_listener "rag4financenew/cdc-listener"
)

// MyEventHandler 负责处理从 Binlog 接收到的事件
type MyEventHandler struct {
	canal.DummyEventHandler // 继承默认的空实现，我们要重写 OnRow
	temporalClient          client.Client
}

// OnRow 当数据行发生变化时（Insert, Update, Delete）会被调用
func (h *MyEventHandler) OnRow(e *canal.RowsEvent) error {
	// e.Rows 里的数据是原始数组，e.Table.Columns 存了列名信息
	// 我们要把它们“拉链”到一起

	// 遍历所有变动行
	for i, row := range e.Rows {
		// 处理 Update 事件的特殊逻辑
		// Update 事件在 e.Rows 里是成对出现的：[旧值, 新值, 旧值, 新值...]
		// 我们只需要同步“新值”，所以跳过偶数索引 (0, 2, 4...)
		if e.Action == canal.UpdateAction && i%2 == 0 {
			continue
		}

		// 1. 数据组装：将 Slice 转为 Map
		rowData := map[string]interface{}{}
		for colIndex, col := range e.Table.Columns {
			// e.Table.Columns[colIndex].Name 是列名
			// row[colIndex] 是对应的值
			rowData[col.Name] = row[colIndex]
		}

		// 2. 构造快递包裹
		event := cdc_listener.CDCEvent{
			TableName: e.Table.Name,
			Action:    e.Action,
			Data:      rowData,
		}

		// 3. 触发 Temporal Workflow
		// 这里的 ID 用 TableName + PrimaryKey 组合最好，保证防抖和唯一性
		// 假设第一个字段是主键，简单起见我们先用随机或表名
		workflowId := fmt.Sprintf("sync-%s-%v", event.TableName, rowData["id"])

		options := client.StartWorkflowOptions{
			ID:        workflowId,
			TaskQueue: cdc_listener.QueueName,
		}

		we, err := h.temporalClient.ExecuteWorkflow(context.Background(), options, "SyncDataWorkflow", event)
		if err != nil {
			log.Printf("启动 Workflow 失败: %v", err)
			return nil // 不要返回 err，否则 Canal 会停止监听
		}
		fmt.Printf("已触发 Workflow, RunID: %s, 数据: %v\n", we.GetRunID(), rowData["name"]) // 假设有个 name 字段
	}

	return nil
}

func main() {
	// 1. 初始化 Temporal Client
	tc, err := client.Dial(client.Options{
		Namespace: cdc_listener.NameSpace,
	})
	if err != nil {
		log.Fatalln("无法连接 Temporal Server:", err)
	}
	defer tc.Close()
	// 1. 配置 Canal
	cfg := canal.NewDefaultConfig()
	cfg.Addr = "127.0.0.1:3306"
	cfg.User = "root"
	cfg.Password = "rootpassword" // 替换你的密码
	cfg.Dump.ExecutionPath = ""   // 如果不需要全量 dump，可以留空或指向 mysqldump

	// 我们只关心特定的数据库和表
	// 注意：Canal 默认会解析所有表，建议用 IncludeTableRegex 过滤，减少开销
	cfg.IncludeTableRegex = []string{"baoer\\.activity20190301"}

	// 2. 创建 Canal 实例
	c, err := canal.NewCanal(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// 3. 注册我们的处理器
	c.SetEventHandler(&MyEventHandler{temporalClient: tc})

	// 4. 开始运行
	// StartFrom() 会尝试从最新的位置开始监听
	// 实际上生产环境我们需要从上次保存的位置（Position）开始，这里先简化
	fmt.Println("开始监听 MySQL Binlog...")
	if err = c.Run(); err != nil {
		log.Fatal(err)
	}
}
