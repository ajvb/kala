FROM golang

WORKDIR /go/src/github.com/ajvb/kala
COPY . .
RUN go build && mv kala /usr/bin

CMD ["kala", "run"]
EXPOSE 8000
