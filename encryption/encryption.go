package encryption

import (
	"encoding/base64"
	"strings"
)

func DecodeBase(msg, decode string) bool {
	msgDecode, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return false
	}
	if !strings.EqualFold(string(msgDecode), decode) {
		return false
	}
	return true
}

func DecodeByte(msg string) []byte {
	sDec, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return nil
	}
	return sDec
}
