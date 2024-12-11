FROM golang:alpine

USER guest
WORKDIR /home/guest

COPY go.mod go.sum main.go views .
COPY data/assets data/assets

EXPOSE 3000

CMD go run .
