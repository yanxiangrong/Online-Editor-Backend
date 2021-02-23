FROM golang:latest

WORKDIR .
RUN go build .

EXPOSE 9527
ENTRYPOINT ["./mian"]