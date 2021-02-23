FROM golang:latest

WORKDIR $GOPATH/src/onilne-editor-backend
COPY . $GOPATH/src/onilne-editor-backend
RUN go env -w GOPROXY=https://goproxy.cn
RUN go build .

EXPOSE 9527
#ENV GIN_MODE release
ENTRYPOINT ["./online-editor-backend"]
