FROM golang:1.19 AS builder

WORKDIR /user/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
COPY ./tmp /usr/local/bin/tmp

WORKDIR /user/src/app
RUN go build -v -o /usr/local/bin/app ./Api

EXPOSE 8080

CMD ["app"]
