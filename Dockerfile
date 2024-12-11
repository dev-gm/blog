FROM golang:alpine as build

WORKDIR /blog

COPY . .

RUN go mod download && go build -o blog

FROM alpine:latest

USER guest
WORKDIR /home/guest

COPY --from=build /blog/blog ./blog

EXPOSE 8080

CMD ["./blog"]
