FROM golang:latest

WORKDIR $GOPATH/src/Online-Editor-Backend
COPY . $GOPATH/src/Online-Editor-Backend
RUN go env -w GOPROXY=https://goproxy.cn
RUN go build .
RUN apt install python3 openjdk-11-jdk -y

EXPOSE 9527
#ENV GIN_MODE release
ENTRYPOINT ["./online-editor-backend"]
