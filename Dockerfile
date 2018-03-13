FROM golang

RUN go get github.com/ajvb/kala
CMD ["kala", "run"]
EXPOSE 8000
