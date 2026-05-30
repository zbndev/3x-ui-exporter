FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /3x-ui-exporter .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /3x-ui-exporter /3x-ui-exporter
EXPOSE 9847
ENTRYPOINT ["/3x-ui-exporter"]
