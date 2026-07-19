FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/mockserver .

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/mockserver /mockserver
EXPOSE 8080
ENTRYPOINT ["/mockserver"]
