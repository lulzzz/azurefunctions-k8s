FROM golang:1.10.3-alpine3.8

ENV KUBE_LATEST_VERSION="v1.11.1"

RUN apk add --update ca-certificates \
 && apk add --update -t deps curl \
 && apk add --no-cache bash git openssh \
 && curl -L https://storage.googleapis.com/kubernetes-release/release/${KUBE_LATEST_VERSION}/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl \
 && chmod +x /usr/local/bin/kubectl \
 && apk del --purge deps \
 && rm /var/cache/apk/*

COPY dist/azcontroller /go/src/app
WORKDIR /go/src/app
RUN chmod +x ./azcontroller

CMD ["./azcontroller"]