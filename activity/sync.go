package activity

import (
	"context"
	"github.com/qdrant/go-client/qdrant"
	"rag4financenew/client"
	"rag4financenew/util"
)

type SyncActivities struct {
}

func (s *SyncActivities) Qdrant(ctx context.Context, rows *qdrant.PointStruct) error {
	wait := true
	_, err := client.Qdrant.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: util.NewsCollectionName,
		Points:         []*qdrant.PointStruct{rows},
		Wait:           &wait,
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
