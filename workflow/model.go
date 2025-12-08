package workflow

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

type ChatReq struct {
	SessionID  string `json:"session_id"`
	Question   string `json:"question"`
	NewSession bool   `json:"new_session"`
}

func (u *ChatReq) UnmarshalJSON(bytes []byte) error {
	type Alias ChatReq
	tmp := &Alias{}
	if err := json.Unmarshal(bytes, tmp); err != nil {
		return nil
	}
	// ----- 序列化之前的 Hook -----
	if tmp.SessionID == "" {
		tmp.SessionID = uuid.New().String()
	}

	*u = ChatReq(*tmp)
	return nil
}

func (u *ChatReq) GetUpdateName() string {
	return "chat_message"
}

func (u *ChatReq) GetWorkflowID() string {
	return fmt.Sprintf("chat_session_%s", u.SessionID)
}

type ChatResp struct {
	SessionID    string `json:"session_id"`
	ReplyMessage string `json:"reply_message"`
}
