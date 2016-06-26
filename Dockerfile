FROM golang:1.6.2-alpine
RUN apk add --no-cache git

RUN go get github.com/cescoferraro/kala
ENTRYPOINT kala run --jobDB=redis --jobDBAddress=redis.db.svc.cluster.local:6379  --jobDBPassword=descriptor8 -v
EXPOSE 8000
