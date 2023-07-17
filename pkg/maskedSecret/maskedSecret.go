package maskedSecret

type MaskedSecret struct {
	Masked string `json:"masked"`
	Secret string `json:"secret"`
}

type MaskedEntry struct {
	MaskedSecrets []MaskedSecret `json:"maskedSecrets"`
	MaskedFile string `json:"maskedFile"`
}