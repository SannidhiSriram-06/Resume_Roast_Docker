# ── Stage 1: Build ─────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o resume-roaster .

# ── Stage 2: Run ───────────────────────────────────────────────────────────────
FROM debian:bookworm-slim

# Install poppler-utils for pdftotext
RUN apt-get update && apt-get install -y --no-install-recommends \
    poppler-utils \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/resume-roaster .
COPY static/ ./static/

EXPOSE 8080

CMD ["./resume-roaster"]
