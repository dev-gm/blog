FROM node:alpine AS builder

COPY ./web /usr/local/web
WORKDIR /usr/local/web

RUN npm install
RUN npm run build

FROM golang:alpine
USER guest

WORKDIR /home/guest
COPY go.mod .
COPY go.sum .
COPY main.go .
COPY ./views ./views
COPY --from=builder /usr/local/web/dist/index.html ./views/nested/index.html

EXPOSE 3000

CMD ["sh", "-c", "go build . && ./blog"]
