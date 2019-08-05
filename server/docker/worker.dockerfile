# golang 1.12.7-buster
FROM golang@sha256:55803225abf9cdc5b42c913d5d8c8f2add70ae101650d64a5f92fdf685309b5a AS build
WORKDIR /go/src/github.com/abustany/moblog-cloud
COPY . .
RUN GOFLAGS=-mod=vendor CGO_ENABLED=0 GOOS=linux make worker

# alpine 3.10.1
FROM alpine@sha256:6a92cd1fcdc8d8cdec60f33dda4db2cb1fcdcacf3410a8e05b3741f44a9b5998 AS hugo
RUN mkdir /tmp/hugo && cd /tmp/hugo && \
	HUGO_VERSION=0.56.3 && \
	HUGO_TGZ_SHA256=e77aafdb1b9c7442a5c4dd32c03443d8ac578cc838704b975686ec0d87797907 && \
	wget https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_${HUGO_VERSION}_Linux-64bit.tar.gz && \
	echo "${HUGO_TGZ_SHA256}  hugo_${HUGO_VERSION}_Linux-64bit.tar.gz" | sha256sum -c && \
	tar xf hugo_${HUGO_VERSION}_Linux-64bit.tar.gz

# alpine 3.10.1
FROM alpine@sha256:6a92cd1fcdc8d8cdec60f33dda4db2cb1fcdcacf3410a8e05b3741f44a9b5998
RUN apk --no-cache add ca-certificates git
RUN adduser -h /home/worker -D worker
WORKDIR /home/worker
COPY --from=build /go/src/github.com/abustany/moblog-cloud/worker .
COPY --from=hugo /tmp/hugo/hugo /usr/bin
USER worker
CMD ["/home/worker/worker"]
