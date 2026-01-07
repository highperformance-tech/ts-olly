FROM golang:1.25-alpine as build

WORKDIR /go/src
COPY . .

RUN CGO_ENABLED=0 make build/linux && chmod -R +x bin

# Now copy it into our base image.
FROM gcr.io/distroless/static-debian11
COPY --from=build /go/src/bin/linux_amd64/ts-olly /
CMD ["/ts-olly"]