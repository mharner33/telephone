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

var hosts = []string{"tele0", "tele1", "tele2", "tele3", "tele4"}
var hostMsgMap = map[string]string{
	"tele0": "http://tele0:8080/message",
	"tele1": "http://tele1:8081/message",
	"tele2": "http://tele2:8082/message",
	"tele3": "http://tele3:8083/message",
	"tele4": "http://tele4:8084/message",
}
var hostHealthtap = map[string]string{
	"tele0": "http://tele0:8080/health",
	"tele1": "http://tele1:8081/health",
	"tele2": "http://tele2:8082/health",
	"tele3": "http://tele3:8083/health",
	"tele4": "http://tele4:8084/health",
}

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

	//nextServiceURL := os.Getenv("NEXT_SERVICE_URL")
	nextServiceURL := getNextHostURL()
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

func coinFlip() bool {
	return rand.Intn(2) == 1
}

func getNextHost() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Error getting hostname: %v", err)
		return ""
	}

	currentIndex := -1
	for i, host := range hosts {
		if host == hostname {
			currentIndex = i
			break
		}
	}

	// If current hostname not found in array, start from 0
	if currentIndex == -1 {
		log.Printf("Hostname not found in array, starting from 0", currentIndex)
		currentIndex = -1

	}

	// Try each host starting from the next one
	for i := 1; i <= len(hosts); i++ {
		nextIndex := (currentIndex + i) % len(hosts)
		nextHost := hosts[nextIndex]

		// Check health of this host
		if checkHostHealth(nextHost) {
			return nextHost
		}
	}

	// If no healthy host found, return the immediate next host anyway
	nextIndex := (currentIndex + 1) % len(hosts)
	return hosts[nextIndex]
}

func checkHostHealth(host string) bool {
	healthURL, exists := hostHealthtap[host]
	if !exists {
		return false
	}

	resp, err := http.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func getNextHostURL() string {
	nextHost := getNextHost()

	// If next host is the first in the list, we've completed the cycle
	if nextHost == hosts[0] {
		return ""
	}

	if url, exists := hostMsgMap[nextHost]; exists {
		return url
	}
	return ""
}

func getNextHostHealth() bool {
	nextHost := getNextHost()
	healthURL, exists := hostHealthtap[nextHost]
	if !exists {
		log.Printf("Health check failed for %s: host not found in health map", nextHost)
		return false
	}

	resp, err := http.Get(healthURL)
	if err != nil {
		log.Printf("Health check failed for %s: %v", nextHost, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Health check failed for %s: status %d", nextHost, resp.StatusCode)
		return false
	}

	return true
}

func modifyMessage(text string) string {
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
