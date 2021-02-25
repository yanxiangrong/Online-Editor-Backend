FROM golang:latest

WORKDIR $GOPATH/src/Online-Editor-Backend
COPY . $GOPATH/src/Online-Editor-Backend
RUN go env -w GOPROXY=https://goproxy.cn
RUN go build .
RUN apt update
RUN apt install python3 openjdk-11-jdk -y

EXPOSE 9527
#ENV GIN_MODE release
ENV DB_USERNAME OnlineEditor
ENV DB_PASSWD password
ENV DB_ADDR yandage.top
ENV DB_PORT 3306
ENV DB_DBNAME onlineeditor
ENTRYPOINT ["./online-editor-backend"]
