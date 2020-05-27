FROM golang

WORKDIR /go/src/github.com/ajvb/kala
COPY . .
RUN go build && mv kala /usr/bin

CMD ["kala", "serve"]
EXPOSE 8000
