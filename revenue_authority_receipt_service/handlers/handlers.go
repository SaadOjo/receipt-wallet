package handlers

import (
	"net/http"

	"revenue-authority-receipt-service/crypto"
	"revenue-authority-receipt-service/models"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	cryptoService *crypto.CryptoService
}

func NewHandler(cryptoService *crypto.CryptoService) *Handler {
	return &Handler{
		cryptoService: cryptoService,
	}
}

func (h *Handler) SignHash(c *gin.Context) {
	var req models.SignRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid request format",
		})
		return
	}

	signature, err := h.cryptoService.SignHash(req.Hash)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SignResponse{
		Signature: signature,
	})
}

func (h *Handler) GetPublicKey(c *gin.Context) {
	publicKey, err := h.cryptoService.GetPublicKeyBase64()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to retrieve public key",
		})
		return
	}

	c.JSON(http.StatusOK, models.PublicKeyResponse{
		PublicKey: publicKey,
	})
}