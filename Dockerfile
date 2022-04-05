FROM golang:1.18-alpine as builder

WORKDIR /src

RUN apk add --no-cache git

COPY . .

RUN go build ./cmd/lightcable

FROM alpine

COPY --from=builder /src/lightcable /usr/bin/lightcable

EXPOSE 8080/tcp

ENTRYPOINT ["/usr/bin/lightcable"]
