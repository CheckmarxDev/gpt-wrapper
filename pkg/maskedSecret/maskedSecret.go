package maskedSecret

type MaskedSecret struct {
	Masked string `json:"masked"`
	Secret string `json:"secret"`
	Line   int    `json:"line"`
}

type MaskedEntry struct {
	MaskedSecrets []MaskedSecret `json:"maskedSecrets"`
	MaskedFile    string         `json:"maskedFile"`
}
