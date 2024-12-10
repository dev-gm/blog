FROM node:alpine AS builder

WORKDIR /usr/local/blog
COPY ./web .

RUN npm install
RUN npm run build

FROM golang:bookworm

RUN useradd -m blog

USER blog

WORKDIR /home/blog
COPY go.mod .
COPY go.sum .
COPY main.go .
COPY ./views ./views
COPY --from-builder /usr/local/blog/web/dist/index.html ./views/nested/index.html

EXPOSE 3000

CMD ["sh", "-c", "go build . && ./blog"]
