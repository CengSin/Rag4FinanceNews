package workflow

type RagChatWorkflowReq struct {
	SessionId string `json:"session_id"`
	Question  string `json:"question"`
}
