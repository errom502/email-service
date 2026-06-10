package tools

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

type hashTool struct {
	signature []byte
}

func NewHash(signature string) (*hashTool, error) {
	raw, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("app.decodeSignature: %w", err)
	}

	if len(raw) != 32 {
		return nil, fmt.Errorf("app.decodeSignature: signature must be 32 bytes, but got %d", len(raw))
	}

	return &hashTool{
		signature: raw,
	}, nil
}

// BuildVerificationHash создает HMAC-SHA256 хеш на основе verificationID и secret, используя секретную подпись.
func (h *hashTool) BuildVerificationHash(
	verificationID uuid.UUID,
	secret string,
) string {
	msg := fmt.Sprintf("%s:%s", verificationID.String(), secret)

	hash := hmac.New(sha256.New, h.signature)
	hash.Write([]byte(msg))

	return hex.EncodeToString(hash.Sum(nil))
}
