# golang 1.12.7-buster
FROM golang@sha256:55803225abf9cdc5b42c913d5d8c8f2add70ae101650d64a5f92fdf685309b5a AS build
WORKDIR /go/src/github.com/abustany/moblog-cloud
COPY . .
RUN GOFLAGS=-mod=vendor CGO_ENABLED=0 GOOS=linux make gitserver

# alpine 3.10.1
FROM alpine@sha256:6a92cd1fcdc8d8cdec60f33dda4db2cb1fcdcacf3410a8e05b3741f44a9b5998
RUN apk --no-cache add ca-certificates git
RUN adduser -h /home/gitserver -D gitserver
WORKDIR /home/gitserver
COPY --from=build /go/src/github.com/abustany/moblog-cloud/gitserver .
USER gitserver
CMD ["/home/gitserver/gitserver"]
