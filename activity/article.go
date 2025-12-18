package activity

import (
	"context"
	"github.com/PuerkitoBio/goquery" // 需要 go get
	"strings"
)

type ProcessingActivities struct{}

type ChunkResult struct {
	Chunks []string
}

// CleanAndChunk 清洗 HTML 并进行递归切片
func (a *ProcessingActivities) CleanAndChunk(ctx context.Context, htmlContent string, chunkSize int, overlap int) (*ChunkResult, error) {
	// 1. 清洗 HTML (ETL)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err // 或者降级处理，直接当纯文本
	}
	// 提取纯文本，这里可以加更复杂的逻辑，比如保留 <h1> 作为 Metadata
	textContent := doc.Text()
	return a.CleanAndChunkText(ctx, textContent, chunkSize, overlap)
}

func (a *ProcessingActivities) CleanAndChunkText(ctx context.Context, textContent string, chunkSize int, overlap int) (*ChunkResult, error) {
	// 去除多余空行
	textContent = strings.Join(strings.Fields(textContent), " ")

	// 2. 递归切片 (Recursive Chunking)
	// 简单实现逻辑：优先按句号分，再按长度分
	chunks := recursiveSplit(textContent, chunkSize, overlap)

	return &ChunkResult{Chunks: chunks}, nil
}

// recursiveSplit 是一个简化的递归切片实现
func recursiveSplit(text string, limit int, overlap int) []string {
	var chunks []string
	runes := []rune(text)

	if len(runes) <= limit {
		return []string{text}
	}

	// 实际工程中，这里应该用 langchaingo 的 splitter 或者更复杂的正则
	// 这里演示核心逻辑：滑动窗口
	for i := 0; i < len(runes); {
		end := i + limit
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))

		// 移动步长 = 窗口 - 重叠
		i += (limit - overlap)
	}
	return chunks
}
