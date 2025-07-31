package main

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const (
	CharSet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

type Shortner struct {
	Db *gorm.DB
}

func GenerateShortIDWithBase62Encoding(id uint) string {
	var result strings.Builder
	if id == 0 {
		return string(CharSet[id])
	}

	for id > 0 {
		remainder := id % 62
		result.WriteByte(CharSet[remainder])
		id = id / 62
	}
	encoded := []rune(result.String())
	for i, j := 0, len(encoded)-1; i < j; i, j = i+1, j-1 {
		encoded[i], encoded[j] = encoded[j], encoded[i]
	}
	return string(encoded)
}

func Base62Decode(shortID string) (uint, error) {
	var result uint
	// find the index of every char from charset
	for _, c := range shortID {
		index := strings.IndexRune(CharSet, c)
		if index == -1 {
			return 0, fmt.Errorf("Invalid character %v\n.", c)
		}
		result = result*62 + uint(index)
	}
	return result, nil
}
