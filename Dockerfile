FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /app/bin/lintasan ./cmd/lintasan/

FROM alpine:3.20

RUN apk add --no-cache sqlite-libs ca-certificates

WORKDIR /app
COPY --from=builder /app/bin/lintasan /app/lintasan

RUN mkdir -p /app/data

EXPOSE 20180
ENV PORT=20180
ENV LINTASAN_DATA_DIR=/app/data

ENTRYPOINT ["/app/lintasan"]
CMD ["start"]
