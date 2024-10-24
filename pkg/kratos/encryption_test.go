// Copyright 2024 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package kratos

import (
	"encoding/hex"
	"errors"
	"io"
	"testing"

	"go.uber.org/mock/gomock"
)

//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_cipher.go crypto/cipher AEAD
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_tracing.go go.opentelemetry.io/otel/trace Tracer

const (
	mockSecretKey       = "caskjdflasjkfdlaksjfalskdfjasfda"
	mockStringEncrypted = "b44c81a3577085a44105f90902da59a882d5f94f436a5da27ac7c26d7df87cd553d24f845ee4ac"
	mockStringPlain     = "mock-string"
)

func TestEncrypt_Decrypt(t *testing.T) {
	ctrl := gomock.NewController(t)

	tests := []struct {
		name           string
		hexData        string
		setupMocks     func(*Encrypt, *MockLoggerInterface)
		expected       string
		expectedErrMsg string
	}{
		{
			name:           "Success",
			hexData:        mockStringEncrypted,
			expected:       mockStringPlain,
			setupMocks:     func(e *Encrypt, logger *MockLoggerInterface) {},
			expectedErrMsg: "",
		},
		{
			name:     "FailureDecodeHex",
			hexData:  "not-hex-encoded-string",
			expected: "",
			setupMocks: func(e *Encrypt, logger *MockLoggerInterface) {
				logger.EXPECT().Error("error decoding hex encoded string, encoding/hex: invalid byte: U+006E 'n'")
			},
			expectedErrMsg: "error decoding hex encoded string, encoding/hex: invalid byte: U+006E 'n'",
		},
		{
			name:    "FailureDecodedHexTooShort",
			hexData: hex.EncodeToString([]byte("short")),
			setupMocks: func(encrypt *Encrypt, logger *MockLoggerInterface) {
				logger.EXPECT().Error("encrypted data malformed")
			},
			expectedErrMsg: "encrypted data malformed",
		},
		{
			name:     "FailureDecryption",
			hexData:  mockStringEncrypted,
			expected: "",
			setupMocks: func(e *Encrypt, logger *MockLoggerInterface) {
				logger.EXPECT().Error("error decrypting data: mock-error")

				mockGcm := NewMockAEAD(ctrl)
				mockGcm.EXPECT().Open(nil, gomock.Any(), gomock.Any(), nil).
					Times(1).Return(nil, errors.New("mock-error"))
				mockGcm.EXPECT().NonceSize().Times(1).Return(12)
				e.gcm = mockGcm
			},
			expectedErrMsg: "error decrypting data: mock-error",
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			logger := NewMockLoggerInterface(ctrl)
			tracer := NewMockTracingInterface(ctrl)
			e := NewEncrypt([]byte(mockSecretKey), logger, tracer)

			tt.setupMocks(e, logger)

			got, err := e.Decrypt(tt.hexData)

			if (err != nil) && err.Error() != tt.expectedErrMsg {
				t.Errorf("Decrypt() error = %v, expected %v", err, tt.expectedErrMsg)
				return
			}

			if tt.expected != "" && got != tt.expected {
				t.Errorf("Decrypt() got = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestEncrypt_Encrypt(t *testing.T) {
	ctrl := gomock.NewController(t)

	tests := []struct {
		name           string
		plainData      string
		setupMocks     func(*MockLoggerInterface)
		expected       string
		expectedErrMsg string
	}{
		{
			name:      "Success",
			plainData: mockStringPlain,
			expected:  "616161616161616161616161923066289ca92b14015213279526f358a73fd718956ea8eea85c2d",
			setupMocks: func(logger *MockLoggerInterface) {
				ioReadFull = func(r io.Reader, buf []byte) (n int, err error) {
					copy(buf, "aaaaaaaaaaaa")
					return 12, nil
				}
			},
			expectedErrMsg: "",
		},
		{
			name:      "FailureNonceGeneration",
			plainData: mockStringPlain,
			expected:  "",
			setupMocks: func(logger *MockLoggerInterface) {
				ioReadFull = func(r io.Reader, buf []byte) (n int, err error) {
					return 0, errors.New("mock-error")
				}
				logger.EXPECT().Error("error generating random the nonce, mock-error")
			},
			expectedErrMsg: "error generating random the nonce, mock-error",
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				// put things back in order
				ioReadFull = io.ReadFull
			}()

			logger := NewMockLoggerInterface(ctrl)
			tracer := NewMockTracingInterface(ctrl)
			e := NewEncrypt([]byte(mockSecretKey), logger, tracer)

			tt.setupMocks(logger)

			got, err := e.Encrypt(tt.plainData)

			if (err != nil) && err.Error() != tt.expectedErrMsg {
				t.Errorf("Encrypt() error = %v, expected %v", err, tt.expectedErrMsg)
				return
			}

			if tt.expected != "" && got != tt.expected {
				t.Errorf("Encrypt() got = %v, expected %v", got, tt.expected)
			}
		})
	}
}
