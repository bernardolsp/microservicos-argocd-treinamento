# Version 3: Slow - high latency responses
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY ../main.go ../go.mod ../go.sum ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o demo-app .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/demo-app .

ENV VERSION=3.0
ENV BEHAVIOR=slow
ENV PORT=8080

EXPOSE 8080
CMD ["./demo-app"]
