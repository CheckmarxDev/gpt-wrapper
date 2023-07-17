package maskedSecret

type MaskedSecret struct {
	Masked string `json:"masked"`
	Secret string `json:"secret"`
}