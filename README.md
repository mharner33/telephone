# Telephone Game Microservice

This Go microservice emulates the "telephone game". It receives a message, randomly modifies it, and passes it along to the next service in the chain.

## Endpoints

*   `POST /message`: Receives a JSON payload with a message, modifies it, and forwards it.
    *   Request body: `{"text": "your message here"}`
*   `GET /health`: A health check endpoint. Returns `OK`.

## Configuration

The service is configured using environment variables:

*   `PORT`: The port the service listens on. Defaults to `8080`.
*   `NEXT_SERVICE_URL`: The full URL of the next telephone service to call (e.g., `http://telephone-2:8080/message`). If this is not set, the message chain ends.

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
    curl -X POST -d '{"text":"hello world"}' http://localhost:8080/message
    ```

    Check the logs:

    ```bash
    docker logs telephone-1
    ```

3.  **Run multiple instances (a chain):**

    Let's run two services. The first will forward to the second.

    ```bash
    docker run -p 8081:8080 --name telephone-2 -d telephone
    docker run -p 8080:8080 --name telephone-1 -e NEXT_SERVICE_URL=http://host.docker.internal:8081/message -d telephone
    ```
    *Note: `host.docker.internal` is used to allow the container to connect to the host machine.*


    Now, send a message to the first service:

    ```bash
    curl -X POST -d '{"text":"hello world"}' http://localhost:8080/message
    ```

    Check the logs of both containers to see the message being passed and modified:

    ```bash
    docker logs telephone-1
    docker logs telephone-2
    ```

## Running locally

1.  **Run the service:**

    You can set environment variables in the same line:

    ```bash
    go run main.go
    ```

    To run a chain, you'll need multiple terminals.

    *Terminal 1:*
    ```bash
    go run main.go
    ```

    *Terminal 2:*
    ```bash
    PORT=8081 NEXT_SERVICE_URL=http://localhost:8080/message go run main.go
    ```
