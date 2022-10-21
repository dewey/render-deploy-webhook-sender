FROM golang:1.19-alpine as builder

RUN apk add git bash

ENV GO111MODULE=on

# Add our code
ADD ./ $GOPATH/src/github.com/dewey/render-deploy-webhook-sender

# build
WORKDIR $GOPATH/src/github.com/dewey/render-deploy-webhook-sender
RUN cd $GOPATH/src/github.com/dewey/render-deploy-webhook-sender && \    
    GO111MODULE=on GOGC=off go build -mod=vendor -v -o /render-deploy-webhook-sender .

# multistage
FROM alpine:latest

# https://stackoverflow.com/questions/33353532/does-alpine-linux-handle-certs-differently-than-busybox#33353762
RUN apk --update upgrade && \
    apk add curl ca-certificates && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*

COPY --from=builder /render-deploy-webhook-sender /usr/bin/render-deploy-webhook-sender

# Run the image as a non-root user
RUN adduser -D whr
RUN chmod 0755 /usr/bin/render-deploy-webhook-sender

USER whr

# Run the app. CMD is required to run on Heroku
CMD render-deploy-webhook-sender 