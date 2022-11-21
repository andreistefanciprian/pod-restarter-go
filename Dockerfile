## Build
FROM golang:1.19-buster AS build

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -a -o pod-restarter

## Deploy
FROM gcr.io/distroless/base-debian11

COPY --from=build /app/pod-restarter .

USER nonroot:nonroot

ENTRYPOINT ["./pod-restarter"]