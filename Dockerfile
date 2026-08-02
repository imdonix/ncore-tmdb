FROM oven/bun:1 AS frontend

WORKDIR /app

COPY webapp/package.json webapp/bun.lock* ./webapp/
COPY widget/package.json widget/bun.lock* ./widget/

RUN cd webapp && bun install --frozen-lockfile || bun install
RUN cd widget && bun install --frozen-lockfile || bun install

COPY webapp ./webapp
COPY widget ./widget

RUN mkdir -p internal/static/webapp internal/static/widget
RUN cd webapp && bun run build
RUN cd widget && bun run build


FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /app/internal/static ./internal/static

RUN mkdir -p bin
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/ncore-tmdb ./cmd/run


FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

ENV GIN_MODE=release

COPY --from=builder /app/bin/ncore-tmdb .

RUN mkdir /app/data

EXPOSE 8080

CMD ["./ncore-tmdb"]
