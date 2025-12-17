package activity

import (
	"context"
	"github.com/qdrant/go-client/qdrant"
	"rag4financenew/client"
	"rag4financenew/util"
)

type SyncActivities struct {
}

func (s *SyncActivities) QdrantUpsert(ctx context.Context, rows *qdrant.PointStruct) error {
	wait := true
	_, err := client.Qdrant.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: util.NewsCollectionName,
		Points:         []*qdrant.PointStruct{rows},
		Wait:           &wait,
	})
	return err
}

type UpsertRow struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

type QdrantUpsertReq struct {
	Rows    []*UpsertRow
	ColName string
}

func (s *SyncActivities) QdrantBatchUpsertToCollection(ctx context.Context, req *QdrantUpsertReq) error {
	var points []*qdrant.PointStruct
	for _, row := range req.Rows {
		point := &qdrant.PointStruct{
			Id:      qdrant.NewID(row.ID),
			Vectors: qdrant.NewVectors(row.Vector...),
			Payload: qdrant.NewValueMap(row.Payload),
		}
		points = append(points, point)
	}

	wait := true
	_, err := client.Qdrant.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: req.ColName,
		Points:         points,
		Wait:           &wait,
	})
	return err
}

func (s *SyncActivities) QdrantDelete(ctx context.Context, id string) error {
	wait := true
	// Qdrant 删除操作
	// 注意：Qdrant 的 ID 必须和写入时的一致（UUID 或 Int）。
	// 写入时您用了 uuid.New() 生成 PointID，但在 Payload 里存了 article_id。
	// 这是一个设计上的小坑：如果用随机 UUID 做 PointID，删除时你就需要先根据 Payload(article_id) 查出 PointID，然后再删。

	// 方案 A：使用 Filter 删除 (推荐，简单)
	_, err := client.Qdrant.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: util.NewsCollectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: &qdrant.Filter{
					Must: []*qdrant.Condition{
						{
							ConditionOneOf: &qdrant.Condition_Field{
								Field: &qdrant.FieldCondition{
									Key: "id",
									Match: &qdrant.Match{
										MatchValue: &qdrant.Match_Text{Text: id},
									},
								},
							},
						},
					},
				},
			},
		},
		Wait: &wait,
	})

	return err
}

func (s *SyncActivities) QdrantQueryOrConstruct(ctx context.Context, vector []float32) (*qdrant.PointId, error) {
	searchResult, err := client.Qdrant.Query(ctx, &qdrant.QueryPoints{
		CollectionName: util.NewsCollectionName,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &[]uint64{1}[0],
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(true),
	})
	if err != nil {
		return nil, err
	}
	if len(searchResult) > 0 && searchResult[0].Score > 0.9 {
		point := searchResult[0]
		return point.Id, nil
	}
	return nil, nil
}
