# alpine 3.10.1
FROM alpine@sha256:6a92cd1fcdc8d8cdec60f33dda4db2cb1fcdcacf3410a8e05b3741f44a9b5998 AS build
WORKDIR /build
COPY . .
RUN apk --no-cache add ca-certificates yarn
RUN yarn install && PUBLIC_PATH="/admin/" yarn run build

# alpine 3.10.1
FROM alpine@sha256:6a92cd1fcdc8d8cdec60f33dda4db2cb1fcdcacf3410a8e05b3741f44a9b5998
RUN mkdir /app
COPY --from=build /build/dist/ /app/
