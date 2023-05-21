FROM golang:alpine
WORKDIR /app
COPY . .
RUN apk add tzdata && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata
RUN go build main.go
CMD ["ping 127.0.0.1"]