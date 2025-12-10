package cdc_listener

// CDCEvent 定义了从 MySQL 捕获的一个事件
type CDCEvent struct {
	TableName string                 `json:"table_name"` // 表名
	Action    string                 `json:"action"`     // insert, update
	Data      map[string]interface{} // 关键！我们将数组转为 map: {"id": 1, "content": "hello"}
}

// QueueName 定义 Temporal 任务队列的名称
const QueueName = "CDC_SYNC_QUEUE"

// NameSpace 定义 Temporal 空间的名称
const NameSpace = "CDC_SYNC_NAMESPACE"
