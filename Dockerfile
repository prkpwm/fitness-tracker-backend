FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/fitness_data ./fitness_data
# Copy .env if it exists (for local development)
COPY --from=builder /app/.env* ./

EXPOSE 8080
CMD ["./main"]