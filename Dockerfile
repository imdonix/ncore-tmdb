FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

RUN apk add --no-cache make

COPY . .

RUN make build


FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

ENV GIN_MODE=release

COPY --from=builder /app/bin/ncore-tmdb .
COPY --from=builder /app/widget ./widget

RUN mkdir /app/data

EXPOSE 8080

CMD ["./ncore-tmdb"]
