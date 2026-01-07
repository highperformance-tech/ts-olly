FROM gcr.io/distroless/static-debian12
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/ts-olly /
CMD ["/ts-olly"]
