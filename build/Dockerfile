# build stage
FROM golang:1.24 AS builder

WORKDIR /app

COPY ./go.mod .
COPY ./go.sum .

RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dungeon ./cmd/dungeon.go

# final stage
FROM busybox
WORKDIR /app
COPY --from=builder /app/dungeon /app/
COPY public /app/public
COPY version /app/version
RUN sed -i -e 's#/ws#wss://dungeon.getid.org/ws#g' /app/public/index.html
RUN VERSION=$(cat /app/version) && sed -i -e "s#VERSION#$VERSION#g" /app/public/index.html
EXPOSE 9001
ENTRYPOINT ["./dungeon", "--addr=:9001"]
