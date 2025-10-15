package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/mharner33/telephone/hosts"
	"github.com/mharner33/telephone/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("telephone-game")

type Message struct {
	OriginalText string `json:"original_text"`
	ModifiedText string `json:"modified_text"`
}

// MessageHandler handles POST /message requests
func MessageHandler(w http.ResponseWriter, r *http.Request) {
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

	var originalText, modifiedText string

	// If modified message is blank, this is the first host
	if msg.ModifiedText == "" {
		originalText = msg.OriginalText
		modifiedText = message.Modify(ctx, msg.OriginalText)
		log.Printf("First host - Original message: %s", originalText)
		log.Printf("Modified message: %s", modifiedText)
	} else {
		originalText = msg.OriginalText
		modifiedText = message.Modify(ctx, msg.ModifiedText)
		log.Printf("Original message: %s", originalText)
		log.Printf("Previous modified: %s", msg.ModifiedText)
		log.Printf("New modified message: %s", modifiedText)
	}

	span.SetAttributes(
		attribute.String("original.message", originalText),
		attribute.String("modified.message", modifiedText),
	)
	span.AddEvent("Message modified", oteltrace.WithAttributes(attribute.String("modified.message", modifiedText)))

	//nextServiceURL := os.Getenv("NEXT_SERVICE_URL")
	nextServiceURL := hosts.GetNextHostURL()
	if nextServiceURL != "" {
		go forwardMessage(ctx, originalText, modifiedText, nextServiceURL)
	} else {
		log.Println("End of the line. No NEXT_SERVICE_URL configured.")
		span.AddEvent("End of the line")
	}

	io.WriteString(w, "Message received and forwarded (maybe)")
}

// HealthHandler handles GET /health requests
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}

func forwardMessage(ctx context.Context, originalText string, modifiedText string, url string) {
	ctx, span := tracer.Start(ctx, "forward-message")
	defer span.End()

	msg := Message{OriginalText: originalText, ModifiedText: modifiedText}
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

	client := &http.Client{Timeout: 10 * time.Second}
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
