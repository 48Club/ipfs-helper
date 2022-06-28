FROM golang:alpine

RUN apk add --no-cache git curl jq make gcc musl-dev linux-headers\
    && hash=$(curl https://api.github.com/repos/BNB48Club/ipfs-helper/commits -s | jq -r '.[0].sha')\
    && go install github.com/BNB48Club/ipfs-helper/helper@$hash

FROM alpine

COPY --from=0 /go/bin/helper /usr/bin

ENTRYPOINT [ "helper" ]