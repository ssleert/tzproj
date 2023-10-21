FROM golang:1.20.4 as builder

WORKDIR /app
COPY ./go.mod ./go.sum ./
RUN go mod download

COPY . .
RUN go build


FROM debian:bookworm

WORKDIR /app
COPY --from=builder /app/tzproj /app/app

ENTRYPOINT ["./app"]
