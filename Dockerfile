FROM node:alpine AS builder

COPY web /usr/local/web
WORKDIR /usr/local/web

RUN npm install && npm run build


FROM golang:alpine

USER guest
WORKDIR /home/guest

COPY go.mod go.sum main.go views .
COPY --from=builder /usr/local/web/dist/index.html views/nested/index.html

EXPOSE 3000

CMD go build . && ./blog
