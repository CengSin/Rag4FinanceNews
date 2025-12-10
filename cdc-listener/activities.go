package cdc_listener

import (
	"context"
	"fmt"
	"go.temporal.io/sdk/activity"
)

// 模拟 Embedding 函数 (实际项目中请替换为调用 OpenAI/McpTools)
func MockEmbedding(text string) ([]float32, error) {
	// 假设返回一个 4 维向量
	return []float32{0.1, 0.2, 0.3, 0.4}, nil
}

// Activity: 处理并存储数据
type QdrantActivities struct {
	// 这里可以持有 Qdrant 的 Client 连接
}

func (a *QdrantActivities) SyncToQdrant(ctx context.Context, event CDCEvent) error {
	logger := activity.GetLogger(ctx)
	logger.Info("开始处理数据同步任务", "Table", event.TableName)

	// 1. 提取需要向量化的文本
	// 假设我们约定：数据库里有个字段叫 "intro" 或者 "content" 需要被搜索
	// 实际代码中，你可以把整行数据转成 JSON 字符串，或者指定特定字段
	textToIndex := ""
	if val, ok := event.Data["intro"]; ok {
		textToIndex = fmt.Sprintf("%v", val)
	} else {
		// 如果没找到指定字段，就跳过或者把整个 Map 转 String
		logger.Info("未找到可向量化的字段，跳过")
		return nil
	}

	vector, err := MockEmbedding(textToIndex)
	if err != nil {
		return err
	}

	// 3. 存入 Qdrant
	// 这里是伪代码，展示逻辑。请使用你项目中已有的 Qdrant Client
	/*
		point := &pb.PointStruct{
			Id:   PointID(event.Data["id"]), // 使用数据库 ID 作为 Qdrant ID
			Vectors: &pb.Vectors{
				Vector: &pb.Vector{Data: vector},
			},
			Payload: mapToPayload(event.Data), // 把原始数据存为 Payload，方便检索时直接展示
		}

		_, err = qdrantClient.Upsert(ctx, &pb.UpsertPoints{
			CollectionName: "knowledge_base",
			Points:         []*pb.PointStruct{point},
		})
	*/

	// 模拟打印，代表入库成功
	logger.Info("成功存入 Qdrant!", "Text", textToIndex, "Vector", vector)
	return nil
}
