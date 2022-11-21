## Build
FROM golang:1.19.1-alpine AS build

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o pod-restarter

## Deploy
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /app/pod-restarter /pod-restarter

USER nonroot:nonroot

ENTRYPOINT ["/pod-restarter"]