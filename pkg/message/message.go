package message

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MetaData struct {
	TenantID  string
	RequestID string
	UserAgent string
	Feature   string
}
