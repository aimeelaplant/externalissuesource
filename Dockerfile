FROM golang:1.10-alpine3.7

RUN apk --update upgrade && \
    apk add curl tzdata ca-certificates git && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*

RUN go version

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN dep version

RUN mkdir /gocode/

ENV GOPATH /gocode/

RUN go install github.com/golang/mock/mockgen
