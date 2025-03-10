FROM golang:1.24-alpine

WORKDIR /app

COPY . .

RUN go build -o gochat.bin

CMD ["./gochat.bin"]