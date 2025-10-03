package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("telephone-game")

type Message struct {
	Text string `json:"text"`
}

func newExporter(w io.Writer) (trace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		// Use human-readable output.
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithoutTimestamps(),
	)
}

func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("telephone"),
		),
	)
	return r
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Set up OpenTelemetry
	exporter, err := newExporter(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(newResource()),
	)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	http.HandleFunc("/message", messageHandler)
	http.HandleFunc("/health", healthHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func messageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is accepted", http.StatusMethodNotAllowed)
		return
	}

	// Extract trace context from headers
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	ctx, span := tracer.Start(ctx, "receive-message")
	defer span.End()

	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Received message: %s", msg.Text)
	span.SetAttributes(attribute.String("received.message", msg.Text))

	modifiedText := modifyMessage(msg.Text)
	log.Printf("Modified message: %s", modifiedText)
	span.AddEvent("Message modified", oteltrace.WithAttributes(attribute.String("modified.message", modifiedText)))

	nextServiceURL := os.Getenv("NEXT_SERVICE_URL")
	if nextServiceURL != "" {
		go forwardMessage(ctx, modifiedText, nextServiceURL)
	} else {
		log.Println("End of the line. No NEXT_SERVICE_URL configured.")
		span.AddEvent("End of the line")
	}

	fmt.Fprintf(w, "Message received and forwarded (maybe)")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func modifyMessage(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}
	// Modify one character at random
	if len(runes) > 0 {
		randomIndex := rand.Intn(len(runes))
		randomChar := runes[randomIndex]
		// simple modification: increment the character
		runes[randomIndex] = randomChar + 1
	}

	return string(runes)
}

func forwardMessage(ctx context.Context, text string, url string) {
	ctx, span := tracer.Start(ctx, "forward-message")
	defer span.End()

	msg := Message{Text: text}
	body, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshalling message: %v", err)
		span.RecordError(err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		span.RecordError(err)
		return
	}

	// Inject trace context into headers
	req.Header.Set("Content-Type", "application/json")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error forwarding message: %v", err)
		span.RecordError(err)
		return
	}
	defer resp.Body.Close()

	log.Printf("Forwarded message to %s, status: %s", url, resp.Status)
	span.SetAttributes(attribute.String("forward.url", url), attribute.String("forward.status", resp.Status))
}