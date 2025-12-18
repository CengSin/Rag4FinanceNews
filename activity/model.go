package activity

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/sashabaranov/go-openai"
	"strings"
)

type InputArticle struct {
	ID        string
	Title     string
	Content   string
	ColumnID  string   // 栏目ID
	Tags      []string // 原始标签
	PublishTs int64    // 发布时间戳
	URL       string
}

// ProcessedArticle 是处理后的文章，准备入库
type ProcessedArticle struct {
	InputArticle
	Summary string    // LLM 生成的摘要
	Vector  []float32 // Embedding 向量
}

type CallToolParams struct {
	ToolName  string
	Arguments string
}

type ChatWithLLMParams struct {
	Messages       []openai.ChatCompletionMessage
	Tools          []openai.Tool
	ModelName      string
	ResponseFormat *openai.ChatCompletionResponseFormat
}

type MessagesReq struct {
	SessionId string `json:"session_id"`
	Question  string `json:"question"`
	Update    bool   `json:"-"`
}

type UpdateMessagesReq struct {
	SessionId string
	Messages  []openai.ChatCompletionMessage
}

type TypeNameEnum string

const (
	News     TypeNameEnum = "news"
	Articles TypeNameEnum = "articles"
)

type CDCEvent interface {
	ID() int64
	Table() string
	Type() TypeNameEnum
	TextToIndex() string
	ToPayload() (map[string]any, error)
}

// ContentMessageEvent 针对 content_messages 业务表的 CDC 事件
type ContentMessageEvent struct {
	TableName string
	Action    string
	TypeName  TypeNameEnum

	Id        int64
	Title     string
	Summary   string
	CreatedAt int64
	CreatedBy int64
	IsDeleted int8
	Score     int64
}

func (e *ContentMessageEvent) ID() int64 { return e.Id }

func (e *ContentMessageEvent) Table() string      { return e.TableName }
func (e *ContentMessageEvent) Type() TypeNameEnum { return e.TypeName }

func (e *ContentMessageEvent) TextToIndex() string {
	return fmt.Sprintf("%s\n%s", e.Title, e.Summary)
}

func (e *ContentMessageEvent) ToPayload() (map[string]any, error) {
	// 过滤不符合条件的事件
	isDeleteAction := e.Action == "delete" || e.IsDeleted == 1
	if isDeleteAction {
		if e.Score != 2 {
			return nil, nil
		}
		if e.CreatedBy != 1160671 {
			return nil, nil
		}
	}

	payload := map[string]any{
		"id":         fmt.Sprint(e.Id),
		"table_name": e.TableName,
		"action":     e.Action,
		"is_deleted": e.IsDeleted,
	}

	// 只有非删除操作才需要 embedding 的文本
	if !isDeleteAction {
		text := e.TextToIndex()
		if text == "" {
			return nil, errors.New("text to index is empty")
		}

		payload["title"] = e.Title
		payload["summary"] = e.Summary
		payload["type_name"] = e.TypeName
		payload["created_at"] = e.CreatedAt
		payload["textToIndex"] = text
	}

	return payload, nil
}

// CdcEnvelope 用于跨进程传输 CDC 事件，避免接口无法反序列化的问题。
// Type 指示具体业务类型，Payload 是业务事件的 JSON。
type CdcEnvelope struct {
	Type    TypeNameEnum    `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Table   string          `json:"table,omitempty"`
	Action  string          `json:"action,omitempty"`
}

type RouterIntent struct {
	// 让 LLM 只有两个选择：要么选一个具体的工具名，要么选 "general_chat"
	ToolName  string `json:"tool_name" jsonschema_description:"最适合解决用户问题的工具名称。如果不需要使用工具（如纯闲聊），请填 'general_chat'"`
	Reasoning string `json:"reasoning" jsonschema_description:"选择该工具的简短理由"`
	ArgsHint  string `json:"args_hint" jsonschema_description:"(可选) 提取用户问题中可能的关键参数，辅助后续步骤"`
}

type ArticleEntryEvent struct {
	TableName string
	Action    string
	TypeName  TypeNameEnum
	// 以下字段从 MySQL 表结构映射
	Id                      int64   `json:"id"`                        // 主键
	Title                   string  `json:"title"`                     // 标题
	ContentShort            string  `json:"content_short"`             // 短内容
	Content                 string  `json:"content"`                   // 全文内容
	Status                  string  `json:"status"`                    // 状态
	Precedence              int64   `json:"precedence"`                // 优先级
	Pageviews               int64   `json:"pageviews"`                 // 浏览量
	CommentDisabled         bool    `json:"comment_disabled"`          // 是否禁评
	SourceURI               string  `json:"source_uri"`                // 来源链接
	SourceName              string  `json:"source_name"`               // 来源名称
	ImageURI                string  `json:"image_uri"`                 // 图片链接
	Position                int64   `json:"position"`                  // 位置
	DisplayUserID           int64   `json:"display_user_id"`           // 展示用户ID
	DisplayTime             int64   `json:"display_time"`              // 展示时间（Unix 时间戳）
	IsPriced                bool    `json:"is_priced"`                 // 是否付费
	IsTrail                 bool    `json:"is_trail"`                  // 是否试看
	External                bool    `json:"external"`                  // 是否外部
	CreatedBy               int64   `json:"created_by"`                // 创建者ID
	UpdatedBy               int64   `json:"updated_by"`                // 更新者ID
	CreatedAt               int64   `json:"created_at"`                // 创建时间（Unix 时间戳）
	UpdatedAt               int64   `json:"updated_at"`                // 更新时间（Unix 时间戳）
	Platforms               string  `json:"platforms"`                 // 平台
	Categories              string  `json:"categories"`                // 分类
	Tags                    string  `json:"tags"`                      // 标签（逗号分隔）
	Symbols                 string  `json:"symbols"`                   // 符号
	ContentPreview          string  `json:"content_preview"`           // 内容预览
	ContentParsed           string  `json:"content_parsed"`            // 解析后内容
	ContentArgs             string  `json:"content_args"`              // 内容参数
	IsDeleted               bool    `json:"is_deleted"`                // 是否删除
	Extra                   string  `json:"extra"`                     // 额外字段
	Score                   float64 `json:"score"`                     // 得分
	WordsCount              int64   `json:"words_count"`               // 字数统计
	UnshowContentShort      bool    `json:"unshow_content_short"`      // 是否隐藏短内容
	Themes                  string  `json:"themes"`                    // 主题
	BaoerID                 int64   `json:"baoer_id"`                  // baoer id
	CrawlerSourceID         int64   `json:"crawler_source_id"`         // 爬虫来源ID
	Plates                  string  `json:"plates"`                    // 版块
	IsContentCleaning       *bool   `json:"is_content_cleaning"`       // 内容是否清洗（可空）
	CrawlerWechatID         int64   `json:"crawler_wechat_id"`         // 爬虫微信ID
	ImageURIs               string  `json:"image_uris"`                // 多图链接，可能逗号分隔
	InfluenceScore          int64   `json:"influence_score"`           // 影响力分数
	CustomTag               string  `json:"custom_tag"`                // 自定义标签
	AssetTags               string  `json:"asset_tags"`                // 资产标签
	ImageScore              int64   `json:"image_score"`               // 图片评分
	Funds                   string  `json:"funds"`                     // 资金相关字段
	Subtitle                string  `json:"subtitle"`                  // 副标题
	ShowOnWscn              bool    `json:"show_on_wscn"`              // 是否在 wscn 展示
	Image                   string  `json:"image"`                     // 主图
	Images                  string  `json:"images"`                    // 多图 JSON 或文本
	LimitedTime             int64   `json:"limited_time"`              // 限时（秒）
	HasTransferAudio        bool    `json:"has_transfer_audio"`        // 是否有音频
	ShowLike                bool    `json:"show_like"`                 // 是否显示点赞
	PreviewParsed           string  `json:"preview_parsed"`            // 预览解析内容
	PreviewArgs             string  `json:"preview_args"`              // 预览参数
	ResponsibleEditorUserID string  `json:"responsible_editor_userid"` // 责任编辑用户ID
	HangDown                string  `json:"hang_down"`                 // hang_down 字段
	References              string  `json:"references"`                // 参考来源
}

func (a *ArticleEntryEvent) ID() int64 {
	return a.Id
}

func (a *ArticleEntryEvent) Table() string {
	return "article_entries"
}

func (a *ArticleEntryEvent) Type() TypeNameEnum {
	return Articles
}

func (a *ArticleEntryEvent) TextToIndex() string {
	reader := strings.NewReader(a.Content)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s\n%s\n%s", a.Title, a.ContentShort, doc.Text())
}

func (a *ArticleEntryEvent) ToPayload() (map[string]any, error) {
	// 过滤不符合条件的事件
	isDeleteAction := a.Action == "delete" || a.IsDeleted == true

	payload := map[string]any{
		"id":         fmt.Sprint(a.Id),
		"table_name": a.TableName,
		"action":     a.Action,
		"is_deleted": a.IsDeleted,
	}

	// 只有非删除操作才需要 embedding 的文本
	if !isDeleteAction {
		text := a.TextToIndex()
		if text == "" {
			return nil, errors.New("text to index is empty")
		}

		payload["title"] = a.Title
		payload["summary"] = a.ContentShort
		payload["type_name"] = a.TypeName
		payload["created_at"] = a.CreatedAt
		payload["textToIndex"] = text
	}

	return payload, nil
}
