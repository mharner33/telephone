package main

import (
	"log"
	"net/http"
	"os"
	"time"

	ddotel "github.com/DataDog/dd-trace-go/v2/ddtrace/opentelemetry"
	"github.com/mharner33/telephone/handlers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func main() {
	provider := ddotel.NewTracerProvider()
	defer provider.Shutdown()
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

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
