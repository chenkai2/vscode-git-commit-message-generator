package i18n

import (
	"fmt"
	"os"
	"strings"
)

// Language represents the supported languages
type Language string

const (
	ZhHans Language = "zh-hans"
	EnUS   Language = "en-us"
)

var currentLanguage = ZhHans

// SetLanguage sets the current language
func SetLanguage(lang Language) {
	currentLanguage = lang
}

// GetLanguage returns the current language
func GetLanguage() Language {
	return currentLanguage
}

// T translates a message by key with optional format arguments
func T(key string, args ...interface{}) string {
	for _, msg := range Messages {
		if msg.Key == key {
			var text string
			switch currentLanguage {
			case ZhHans:
				text = msg.ZhHans
			case EnUS:
				text = msg.EnUS
			default:
				text = msg.EnUS
			}
			if len(args) > 0 {
				return fmt.Sprintf(text, args...)
			}
			return text
		}
	}
	return key
}

// init initializes the language setting based on environment
func init() {
	// Check LANG environment variable
	lang := os.Getenv("LANG")
	lang = strings.ToLower(lang)

	// Set language based on environment
	if strings.HasPrefix(lang, "zh") {
		SetLanguage(ZhHans)
	} else {
		SetLanguage(EnUS)
	}
}
