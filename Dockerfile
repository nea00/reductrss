FROM golang:1.23.4 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o reductrss .

FROM alpine:latest
RUN apk add --no-cache curl
WORKDIR /app
COPY --from=builder /app/reductrss .
RUN echo ' @hourly /app/reductrss' >> /etc/crontabs/root
WORKDIR /app/data
CMD ["crond", "-f"]