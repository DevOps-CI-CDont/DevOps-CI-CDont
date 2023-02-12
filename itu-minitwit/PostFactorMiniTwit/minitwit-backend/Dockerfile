FROM golang:1.19 AS builder

WORKDIR /user/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/minitwitbackend ./Api

EXPOSE 8080

CMD ["minitwitbackend"]
