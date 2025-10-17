package message

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("telephone-game")

// useOllama controls which LLM backend is used.
// When true, Modify will use Ollama; when false, it will use Gemini.
var useOllama = true

// SetUseOllama allows the application to select Ollama vs Gemini at runtime.
func SetUseOllama(v bool) {
	useOllama = v
}

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

	// Call selected LLM backend
	var (
		oppositeWord string
		err          error
	)
	if useOllama {
		oppositeWord, err = callOllamaFunc(ctx, prompt)
	} else {
		oppositeWord, err = callGeminiFunc(ctx, prompt)
	}
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
var callGeminiFunc = callGemini

func callGemini(ctx context.Context, prompt string) (string, error) {
	ctx, span := tracer.Start(ctx, "call-gemini")
	defer span.End()

	span.SetAttributes(attribute.String("gemini.prompt", prompt))

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		// Missing API key; behave like callOllama error path and bubble up
		err := fmt.Errorf("missing GOOGLE_API_KEY or GEMINI_API_KEY")
		span.RecordError(err)
		return "", err
	}

	// Build request to Gemini GenerateContent API
	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role":  "user",
				"parts": []map[string]interface{}{{"text": prompt}},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.2,
			"maxOutputTokens": 8,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		span.RecordError(err)
		return "", err
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent?key=" + apiKey
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
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
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		span.RecordError(err)
		return "", err
	}

	// Extract first text part, mirroring the cleanup in callOllama
	var response string
	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		response = strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)
	}
	span.SetAttributes(attribute.String("gemini.response", response))
	return response, nil
}

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
