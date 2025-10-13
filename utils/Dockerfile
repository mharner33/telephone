# Stage 1: Build dependencies
FROM python:3.11-slim-bookworm as builder

# Install uv
COPY --from=ghcr.io/astral-sh/uv:0.1.34 /uv /usr/local/bin/uv

# Set environment variables for uv
ENV UV_COMPILE_BYTECODE=1 \
    UV_LINK_MODE=copy

# Set the working directory
WORKDIR /app

# Copy pyproject.toml and uv.lock for dependency installation
COPY pyproject.toml uv.lock ./

# Install dependencies using uv sync with caching
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-install-project --no-dev

# Stage 2: Final image
FROM python:3.11-slim-bookworm

# Set the working directory
WORKDIR /app

# Copy the installed dependencies from the builder stage
COPY --from=builder /app/.venv /app/.venv

# Add the virtual environment to the PATH
ENV PATH="/app/.venv/bin:$PATH"

# Copy the rest of the application code
COPY . .

# Install the project itself (if it's installable)
RUN uv sync --frozen --no-dev

CMD ["python", "main.py"]