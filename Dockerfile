FROM golang:1.23-alpine AS build
RUN mkdir /build
ADD . /build
WORKDIR /build
RUN go build -o stocker ./cmd/stocker/main.go

FROM alpine:latest AS production
RUN mkdir /app
WORKDIR /app
COPY --from=build /build/stocker .
ENTRYPOINT ["/app/stocker", "server"]
