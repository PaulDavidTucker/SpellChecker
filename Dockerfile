FROM golang:1.26.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /spellcheck ./cmd/server

FROM scratch
COPY --from=builder /spellcheck /spellcheck
COPY --from=builder /app/dictionaries/ /dictionaries/
COPY --from=builder /app/config/ /config/
EXPOSE 8080

ENTRYPOINT ["/spellcheck"]
