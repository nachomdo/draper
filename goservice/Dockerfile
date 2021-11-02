FROM golang:buster as builder 

WORKDIR /go/src
COPY . .
RUN go mod download && go build -o goapi *.go
ENV BOOTSTRAP_SERVERS="broker:29092"
ENTRYPOINT [ "/go/src/goapi" ]
