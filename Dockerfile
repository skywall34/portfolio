# Stage 1: Building the application
FROM golang:1.24.4-alpine AS builder
# Install c compiler
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN  go build -o portfolio

# Stage 2: Run the application
FROM alpine:latest
RUN apk add --no-cache tzdata gcc musl-dev
WORKDIR /root/
COPY --from=builder /app/portfolio .
COPY --from=builder /app/static ./static
COPY --from=builder /app/templates ./templates
EXPOSE 8081
CMD ["./portfolio"]