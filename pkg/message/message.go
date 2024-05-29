package message

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MetaData struct {
	TenantID      string
	RequestID     string
	UserAgent     string
	Feature       string
	ExternalAzure *ExternalAzure
}

type ExternalAzure struct {
	Endpoint string
	ApiKey   string
}
