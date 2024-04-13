FROM golang:1.20 AS build

WORKDIR /usr/local/go/src/app

COPY go.mod go.sum ./

RUN go mod download

COPY . ./


RUN CGO_ENABLED=0 GOOS=linux go build -o /build/app


FROM ubuntu:latest

WORKDIR /app

# COPY .env /.env
COPY --from=build /build/app ./

# EXPOSE 8080
ENTRYPOINT ["./app"]