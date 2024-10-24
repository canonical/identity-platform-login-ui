// Copyright 2024 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package kratos

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
)

// needed for testing purposes
var ioReadFull = io.ReadFull

type Encrypt struct {
	gcm cipher.AEAD

	logger logging.LoggerInterface
	tracer tracing.TracingInterface
}

// Encrypt takes a plain string and returns a hex encoded string
func (e *Encrypt) Encrypt(data string) (string, error) {
	payload := []byte(data)
	nonce, err := e.generateCipherNonce()
	if err != nil {
		err = fmt.Errorf("error generating random the nonce, %v", err)
		e.logger.Error(err.Error())
		return "", err
	}
	ciphertext := e.gcm.Seal(nonce, nonce, payload, nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt takes hex encoded string and returns the decrypted plain string
func (e *Encrypt) Decrypt(hexData string) (string, error) {
	encrypted, err := hex.DecodeString(hexData)
	if err != nil {
		err = fmt.Errorf("error decoding hex encoded string, %v", err)
		e.logger.Error(err.Error())
		return "", err
	}

	noncePart, payloadPart, err := e.splitNonceFromPayload(encrypted)
	if err != nil {
		e.logger.Error(err.Error())
		return "", err
	}

	decryptedData, err := e.gcm.Open(nil, noncePart, payloadPart, nil)
	if err != nil {
		err = fmt.Errorf("error decrypting data: %v", err)
		e.logger.Error(err.Error())
		return "", err
	}

	return string(decryptedData), nil
}

func (e *Encrypt) splitNonceFromPayload(encrypted []byte) ([]byte, []byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(encrypted) <= nonceSize {
		return nil, nil, fmt.Errorf("encrypted data malformed")
	}

	noncePart, payloadPart := encrypted[:nonceSize], encrypted[nonceSize:]
	return noncePart, payloadPart, nil
}

func (e *Encrypt) generateCipherNonce() ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := ioReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return nonce, nil
}

func NewEncrypt(secretKey []byte, logger logging.LoggerInterface, tracer tracing.TracingInterface) *Encrypt {
	e := new(Encrypt)
	c, err := aes.NewCipher(secretKey)
	if err != nil {
		logger.Fatalf("fatal error creating cipher from secret key, %v", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		logger.Fatalf("fatal error creating gcm from cipher, %v", err)
	}
	e.gcm = gcm

	e.logger = logger
	e.tracer = tracer
	return e
}
