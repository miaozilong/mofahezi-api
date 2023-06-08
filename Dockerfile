FROM golang:alpine
WORKDIR /app
COPY . .
RUN apk add tzdata && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata
RUN go build -o bin/main.out main.go
CMD ["./bin/main.out"]