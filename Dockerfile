# Build etherquery in a stock Go builder container
FROM golang:1.19-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod tidy

COPY . ./

RUN apk add --no-cache gcc make cmake

RUN make build

# Pull etherquery into a second stage deploy alpine container
FROM alpine:latest as prod

WORKDIR /app

COPY --from=build /app/bin/testproject /usr/local/bin

#EXPOSE 8545 8546 30303 30303/udp

ENTRYPOINT testproject