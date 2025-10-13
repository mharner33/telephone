package message

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("telephone-game")

// Modify takes a text string, selects a random word, and replaces it with its opposite using LLM
func Modify(ctx context.Context, text string) string {
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
	randomIndex := randSource(len(words))
	selectedWord := words[randomIndex]

	// Create prompt for LLM
	prompt := "Return only a single word that is the opposite of: " + selectedWord

	// Call Ollama API
	oppositeWord, err := callOllamaFunc(ctx, prompt)
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

// callOllamaFunc is a variable that holds the callOllama function (allows mocking in tests)
var callOllamaFunc = callOllama

// callOllama sends a request to Ollama API and returns the generated text
func callOllama(ctx context.Context, prompt string) (string, error) {
	ctx, span := tracer.Start(ctx, "call-ollama")
	defer span.End()

	span.SetAttributes(attribute.String("ollama.prompt", prompt))
	requestBody := map[string]interface{}{
		"model":  "gemma3:270m",
		"prompt": prompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		span.RecordError(err)
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "http://ollama:11434/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		span.RecordError(err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		span.RecordError(err)
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		span.RecordError(err)
		return "", err
	}

	// Clean up the response - get just the word
	response := strings.TrimSpace(result.Response)
	span.SetAttributes(attribute.String("ollama.response", response))
	return response, nil
}

// randSource is a variable that holds the random source (allows mocking in tests)
var randSource = rand.Intn

func coinFlip() bool {
	return randSource(2) == 1
}
