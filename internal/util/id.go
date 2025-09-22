package util

import (
	"crypto/sha1"
	"encoding/hex"
	"strconv"
)

func GenerateID(file string, start, end int, kind, name string) string {
	base := file + ":" + strconv.Itoa(start) + ":" + strconv.Itoa(end) + ":" + kind + ":" + name
	h := sha1.Sum([]byte(base))
	return hex.EncodeToString(h[:])
}
