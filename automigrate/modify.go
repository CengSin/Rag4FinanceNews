package main

import (
	"context"
	"github.com/qdrant/go-client/qdrant"
)

// 在 client/client.go 或初始化位置调用
func EnsureFullTextIndex(ctx context.Context, client *qdrant.Client) error {
	// 1. 先尝试删除旧的索引 (如果是 keyword)
	wait := true
	_, _ = client.DeleteFieldIndex(ctx, &qdrant.DeleteFieldIndexCollection{
		CollectionName: CollectionName,
		FieldName:      "textToIndex",
		Wait:           &wait,
	})

	// 2. 创建 text 索引
	_, err := client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
		CollectionName: CollectionName,
		FieldName:      "textToIndex",
		FieldType:      qdrant.FieldType_FieldTypeText.Enum(),
		FieldIndexParams: qdrant.NewPayloadIndexParamsText(&qdrant.TextIndexParams{
			Tokenizer: qdrant.TokenizerType_Multilingual, // 关键：支持中文分词
			Lowercase: &[]bool{true}[0],
		}),
	})
	return err
}
