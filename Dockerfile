FROM golang

RUN go get github.com/ajvb/kala
ENTRYPOINT kala run
EXPOSE 8000
