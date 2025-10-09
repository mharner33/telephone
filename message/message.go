package message

import (
	"math/rand"
)

// Modify takes a text string and randomly modifies one character
func Modify(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}

	// Only modify if coin flip is true
	if coinFlip() {
		// Modify one character at random
		if len(runes) > 0 {
			randomIndex := rand.Intn(len(runes))
			randomChar := runes[randomIndex]
			// simple modification: increment the character
			runes[randomIndex] = randomChar + 1
		}
	}

	return string(runes)
}

func coinFlip() bool {
	return rand.Intn(2) == 1
}

