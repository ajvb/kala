FROM golang

WORKDIR /go/src/bitbucket.org/nextiva/nextkala
COPY . .
RUN go build && mv nextkala /usr/bin

CMD ["nextkala", "serve"]
EXPOSE 8000
