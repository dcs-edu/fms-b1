package paystack

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
)

func (c *Client) VerifySignature(body []byte, signatureHeader string) bool {
	hsum, err := hex.DecodeString(signatureHeader)
	if err != nil {
		return false
	}

	mac := hmac.New(sha512.New, []byte(c.secretKey))
	mac.Write(body)
	sum := mac.Sum(nil)

	return hmac.Equal(sum, hsum)
}
