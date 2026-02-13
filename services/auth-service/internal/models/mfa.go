package models

// MFASetupResponse is returned when a user initiates MFA enrollment.
type MFASetupResponse struct {
	Secret    string `json:"secret"`
	QRCodeURL string `json:"qr_code_url"`
	Issuer    string `json:"issuer"`
}

// MFAVerifyRequest is the input for completing MFA verification.
type MFAVerifyRequest struct {
	Code string `json:"code" validate:"required,len=6"`
}
