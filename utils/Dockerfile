# Stage 1: Build dependencies
FROM python:3.12-slim-bookworm as builder

# Install uv
COPY --from=docker.io/astral/uv:latest /uv /uvx /bin/

# Set the working directory
WORKDIR /app

# Copy pyproject.toml and uv.lock for dependency installation
COPY pyproject.toml uv.lock ./

# Install dependencies using uv
RUN uv sync --locked

# Stage 2: Final image
FROM python:3.12-slim-bookworm

# Set the working directory
WORKDIR /app

# Copy the installed dependencies from the builder stage
COPY --from=builder /app/.venv /app/.venv

# Add the virtual environment to the PATH
ENV PATH="/app/.venv/bin:$PATH"

# Copy the rest of the application code
COPY . .

CMD ["python", "main.py"]