FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
COPY . .
RUN go get
RUN go build -o api main.go
RUN go build -o scraper ./cmd/scraper

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/api .
COPY --from=builder /app/scraper .
# this is bad i know
COPY .env . 
RUN ./scraper
EXPOSE 8080
CMD ["./api"]
