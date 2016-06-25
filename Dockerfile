FROM golang

RUN go get github.com/cescoferraro/kala
ENTRYPOINT kala run
EXPOSE 8000
