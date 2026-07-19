FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /out/mockserver .

FROM alpine:3.20
COPY --from=builder /out/mockserver /usr/local/bin/mockserver
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/mockserver"]
