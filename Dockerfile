FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM scratch
COPY --from=builder /app/server /server
COPY templates /templates
CMD ["/server"] 