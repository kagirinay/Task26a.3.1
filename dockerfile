FROM golang:latest AS build
WORKDIR /build
RUN go env -w GO111MODULE=auto
COPY . /build
RUN go install .

FROM alpine:latest
WORKDIR /pipeline
COPY --from=build /build pipeline
ENTRYPOINT ./pipeline
