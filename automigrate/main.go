package main

import (
	"context"
	"fmt"
	"github.com/qdrant/go-client/qdrant"
	"log"
)

// 配置常量
const (
	CollectionName = "fupengshuo_articles"
	VectorSize     = 1536 // 对应 qwen/qwen3-embedding-8b
	QdrantHost     = "localhost"
	QdrantPort     = 6334
)

func main() {
	// 1. 连接 Qdrant
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: QdrantHost,
		Port: QdrantPort,
	})
	if err != nil {
		log.Fatalln("qdrant connection failed, err ", err.Error())
	}
	defer client.Close()

	ctx := context.Background()

	cols, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatalln(err.Error())
	}

	colExist := false
	for _, col := range cols {
		if CollectionName == col {
			colExist = true
			break
		}
	}

	if !colExist {
		// 3. 创建集合

		err = client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: CollectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     VectorSize,
				Distance: qdrant.Distance_Cosine, // 余弦相似度
			}),
		})

		if err != nil {
			log.Fatalln("create collection failed, err ", err)
		}
		// 4. 创建 Payload 索引 (为了高性能过滤)
		// Qdrant 的 Go SDK 稍微有些底层，需要操作 PointsClient
		createIndex(ctx, client, "created_at", qdrant.FieldType_FieldTypeText)
		createIndex(ctx, client, "summary", qdrant.FieldType_FieldTypeKeyword)
		createIndex(ctx, client, "textToIndex", qdrant.FieldType_FieldTypeKeyword)
		createIndex(ctx, client, "title", qdrant.FieldType_FieldTypeKeyword)
	}
}

func createIndex(ctx context.Context, client *qdrant.Client, fieldName string, fieldType qdrant.FieldType) {
	_, err := client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
		CollectionName: CollectionName,
		FieldName:      fieldName,
		FieldType:      &fieldType,
	})
	if err != nil {
		// 忽略"索引已存在"的错误，简化逻辑
		fmt.Printf("⚠️  创建索引 '%s' 时提示 (可能是已存在): %v\n", fieldName, err)
	} else {
		fmt.Printf("✅ 索引 '%s' 创建成功\n", fieldName)
	}
}
