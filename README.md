# Telephone Game Microservice

This Go microservice emulates the "telephone game". It receives a message, randomly modifies it (based on a coin flip), and passes it along to the next service in the chain. The service automatically discovers the next host in a predefined sequence and performs health checks before forwarding messages.

## Endpoints

*   `POST /message`: Receives a JSON payload with a message, modifies it, and forwards it.
    *   Request body: `{"original_text": "your message here", "modified_text": ""}`
    *   For the first host in the chain, `modified_text` should be an empty string
    *   For subsequent hosts, `modified_text` contains the previously modified version
*   `GET /health`: A health check endpoint. Returns `OK`.

## Configuration

The service is configured using environment variables:

*   `PORT`: The port the service listens on. Defaults to `8080`.

## Host Discovery and Health Checking

The service automatically manages a chain of 5 hosts (tele0 through tele4) with the following features:

*   **Automatic Host Discovery**: The service determines the next host in the sequence based on its own hostname
*   **Health Checking**: Before forwarding messages, the service checks the health of the next host
*   **Failover**: If the next host is unhealthy, it automatically tries subsequent hosts in the sequence
*   **Cycle Detection**: When a message completes a full cycle (returns to tele0), forwarding stops

### Host Configuration

The service is pre-configured with the following hosts and their respective ports:
- tele0:8080, tele1:8081, tele2:8082, tele3:8083, tele4:8084

Each host has both message (`/message`) and health (`/health`) endpoints.

## Message Processing

*   **Random Modification**: Messages are only modified if a coin flip returns true (50% chance)
*   **LLM-Based Word Replacement**: When modified, a random word in the message is selected and replaced with its opposite using the Ollama LLM API (gemma3:270m model)
*   The service connects to an Ollama instance at `http://ollama:11434/api/generate`

## OpenTelemetry Tracing

The service is instrumented with OpenTelemetry. By default, it exports traces to the console (stdout), so you will see trace information in the logs when you run the service. This allows you to see the flow of a message across multiple service instances.

Each span in the trace contains attributes about the message being processed and events for when it's modified.

To use a different exporter (like Jaeger or Zipkin), you would modify the `newExporter` function in `main.go`.

## Running with Docker

1.  **Build the Docker image:**

    ```bash
    docker build -t telephone .
    ```

2.  **Run a single instance:**

    This instance will just receive a message and log it, as `NEXT_SERVICE_URL` is not set.

    ```bash
    docker run -p 8080:8080 --name telephone-1 -d telephone
    ```

    Send a message to it:

    ```bash
    curl -X POST -d '{"original_text":"hello world","modified_text":""}' http://localhost:8080/message
    ```

    Check the logs:

    ```bash
    docker logs telephone-1
    ```

3.  **Run multiple instances (a chain):**

    The service now automatically discovers the next host, so you can run multiple instances with different hostnames:

    ```bash
    docker run -p 8080:8080 --name tele0 --hostname tele0 -d telephone
    docker run -p 8081:8080 --name tele1 --hostname tele1 -d telephone
    docker run -p 8082:8080 --name tele2 --hostname tele2 -d telephone
    docker run -p 8083:8080 --name tele3 --hostname tele3 -d telephone
    docker run -p 8084:8080 --name tele4 --hostname tele4 -d telephone
    ```

    Now, send a message to any service (e.g., tele0):

    ```bash
    curl -X POST -d '{"original_text":"hello world","modified_text":""}' http://localhost:8080/message
    ```

    The message will automatically flow through the chain: tele0 → tele1 → tele2 → tele3 → tele4 → (stops)

    Check the logs of any container to see the message flow:

    ```bash
    docker logs tele0
    docker logs tele1
    # ... etc
    ```

## Running locally

1.  **Run the service:**

    You can set environment variables in the same line:

    ```bash
    go run main.go
    ```

    To run a chain locally, you'll need multiple terminals with different hostnames:

    *Terminal 1 (tele0):*
    ```bash
    PORT=8080 go run main.go
    ```

    *Terminal 2 (tele1):*
    ```bash
    PORT=8081 go run main.go
    ```

    *Terminal 3 (tele2):*
    ```bash
    PORT=8082 go run main.go
    ```

    *Terminal 4 (tele3):*
    ```bash
    PORT=8083 go run main.go
    ```

    *Terminal 5 (tele4):*
    ```bash
    PORT=8084 go run main.go
    ```

    Note: For local testing, you may need to modify the hostname detection or use Docker for proper hostname isolation.

## Quote Generator Utility

The `utils/` directory contains a Python-based quote generator that automatically feeds quotes into the telephone chain:

### Features

*   Reads inspirational quotes from `utils/quotes` file (126 quotes)
*   Posts one quote every 15 seconds to the telephone service
*   Cycles through all quotes continuously

### Running the Quote Generator

1.  **Using Docker:**

    Build the Docker image from the utils directory:

    ```bash
    cd utils
    docker build -t telephone-generator .
    ```

    Run the generator (assuming tele0 is accessible):

    ```bash
    docker run -e URL=tele0 -e PORT=8080 telephone-generator
    ```

2.  **Using Python directly:**

    Install dependencies:

    ```bash
    pip install requests
    ```

    Run the script:

    ```bash
    cd utils
    URL=localhost PORT=8080 python main.py
    ```

### Environment Variables

*   `URL`: The hostname or IP address of the telephone service
*   `PORT`: The port of the telephone service
