package im

// ConversationMark represents a conversation mark
type ConversationMark struct {
	AppID               int32  `json:"AppId"`
	ConversationShortID int64  `json:"ConversationShortId"`
	UserID              int64  `json:"UserId"`
	MarkType            int32  `json:"MarkType"`
	MarkValue           string `json:"MarkValue"`
}

// Conversation represents a conversation
type Conversation struct {
	AppID               int32  `json:"AppId"`
	ConversationShortID int64  `json:"ConversationShortId"`
	ConversationType    int32  `json:"ConversationType"`
	Name                string `json:"Name"`
	Description         string `json:"Description"`
	OwnerUserID         int64  `json:"OwnerUserId"`
	CreatedAt           int64  `json:"CreatedAt"`
	UpdatedAt           int64  `json:"UpdatedAt"`
	Ext                 string `json:"Ext"`
}

// Message represents a message
type Message struct {
	AppID               int32  `json:"AppId"`
	ConversationShortID int64  `json:"ConversationShortId"`
	MessageID           string `json:"MessageId"`
	Content             string `json:"Content"`
	ContentType         int32  `json:"ContentType"`
	SenderType          int32  `json:"SenderType"`
	SenderUserID        int64  `json:"SenderUserId"`
	CreatedAt           int64  `json:"CreatedAt"`
	UpdatedAt           int64  `json:"UpdatedAt"`
	Ext                 string `json:"Ext"`
}

// Participant represents a participant
type Participant struct {
	AppID               int32  `json:"AppId"`
	ConversationShortID int64  `json:"ConversationShortId"`
	UserID              int64  `json:"UserId"`
	Role                int32  `json:"Role"`
	JoinedAt            int64  `json:"JoinedAt"`
	Ext                 string `json:"Ext"`
}

// BlockParticipant represents a block participant
type BlockParticipant struct {
	AppID               int32 `json:"AppId"`
	ConversationShortID int64 `json:"ConversationShortId"`
	UserID              int64 `json:"UserId"`
	BlockAction         int32 `json:"BlockAction"`
	BlockTime           int64 `json:"BlockTime"`
	ExpireTime          int64 `json:"ExpireTime"`
}
