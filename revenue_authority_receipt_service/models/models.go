package models

type SignRequest struct {
	Hash string `json:"hash" binding:"required"`
}

type SignResponse struct {
	Signature string `json:"signature"`
}

type PublicKeyResponse struct {
	PublicKey string `json:"public_key"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}