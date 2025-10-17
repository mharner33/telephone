package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	ddotel "github.com/DataDog/dd-trace-go/v2/ddtrace/opentelemetry"
	"github.com/mharner33/telephone/handlers"
	"github.com/mharner33/telephone/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func main() {
	// CLI flag to select LLM provider (ollama or gemini). Default: gemini
	llmProvider := flag.String("llm", "gemini", "LLM provider: 'ollama' or 'gemini'")
	flag.Parse()

	provider := ddotel.NewTracerProvider()
	defer provider.Shutdown()
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// Select LLM backend based on flag
	if *llmProvider == "ollama" {
		message.SetUseOllama(true)
		log.Println("Using LLM provider: ollama")
	} else {
		message.SetUseOllama(false)
		log.Println("Using LLM provider: gemini")
	}

	http.HandleFunc("/message", handlers.MessageHandler)
	http.HandleFunc("/health", handlers.HealthHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           nil,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(server.ListenAndServe())
}

// Removed local handlers; now using handlers package
