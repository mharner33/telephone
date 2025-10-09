package message

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

// Modify takes a text string, selects a random word, and replaces it with its opposite using LLM
func Modify(text string) string {
	// Only modify if coin flip is true
	if !coinFlip() {
		log.Println("No modifications were made")
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		log.Println("No modifications were made")
		return text
	}

	// Select a random word
	randomIndex := rand.Intn(len(words))
	selectedWord := words[randomIndex]

	// Create prompt for LLM
	prompt := "Return only a single word that is the opposite of: " + selectedWord

	// Call Ollama API
	oppositeWord, err := callOllama(prompt)
	if err != nil {
		// If error, return original text
		log.Println("No modifications were made")
		return text
	}

	log.Printf("The word %s was changed to %s", selectedWord, oppositeWord)

	// Replace the word
	words[randomIndex] = oppositeWord
	return strings.Join(words, " ")
}

// callOllama sends a request to Ollama API and returns the generated text
func callOllama(prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model":  "gemma3:270m",
		"prompt": prompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post("http://ollama:11434/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Clean up the response - get just the word
	return strings.TrimSpace(result.Response), nil
}

func coinFlip() bool {
	return rand.Intn(2) == 1
}
