FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go mod tidy && go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/bot ./main.go

FROM alpine:3.20
RUN adduser -D appuser
USER appuser
WORKDIR /home/appuser
COPY --from=builder /bin/bot /usr/local/bin/bot
CMD ["/usr/local/bin/bot"]
