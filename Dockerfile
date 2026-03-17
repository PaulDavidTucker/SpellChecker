FROM golang:1.26.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build the main server binary
RUN CGO_ENABLED=0 go build -o /spellcheck ./cmd/server

# Build the cache warmup tool
RUN CGO_ENABLED=0 go build -o /warmup-cache ./cmd/warmup-cache

FROM golang:1.26.1-alpine AS cache-warmer
WORKDIR /app
COPY --from=builder /app/dictionaries/ /dictionaries/
COPY --from=builder /warmup-cache /warmup-cache

# Build argument to select dictionary size: medium, large, or huge
ARG DICT_SIZE=medium

# Set up the active dictionary name (same as runtime)
RUN cp /dictionaries/en_gb_${DICT_SIZE}.txt /dictionaries/en_gb_active.txt

# Warm up the cache by loading the active dictionary once
# This creates OS-level file cache and our application cache
RUN mkdir -p /tmp/spellcheck-cache && \
    /warmup-cache -dict=/dictionaries/en_gb_active.txt -cache-dir=/tmp/spellcheck-cache

FROM scratch

# Build argument to select dictionary size: medium, large, or huge
# Default is medium (original ~96K words)
ARG DICT_SIZE=medium

COPY --from=builder /spellcheck /spellcheck

# Copy all dictionaries initially, then select which one to use as the active dictionary
COPY --from=builder /app/dictionaries/ /dictionaries/
COPY --from=builder /app/config/ /config/

# Copy the pre-warmed cache from the cache-warmer stage
COPY --from=cache-warmer /tmp/spellcheck-cache/ /tmp/spellcheck-cache/

# Create a copy of the selected dictionary as the active one
# This allows switching between medium, large, and huge without changing code
COPY --from=builder /app/dictionaries/en_gb_${DICT_SIZE}.txt /dictionaries/en_gb_active.txt

EXPOSE 8080

# Environment variables
ENV DICT_PATH=/dictionaries/en_gb_active.txt
ENV CACHE_DIR=/tmp/spellcheck-cache

ENTRYPOINT ["/spellcheck"]
