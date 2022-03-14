FROM golang:1.17.7

WORKDIR /go/src/phantom
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

RUN go build

CMD ["build.sh && build_coinconf.sh"]