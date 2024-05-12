package message

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MetaData struct {
	TenantID    string `json:"tenant_id,omitempty"`
	RequestID   string `json:"request_id"`
	Origin      string `json:"origin"`
	FeatureName string `json:"feature_name"`
	AccessToken string `json:"access_token,omitempty"`
}
