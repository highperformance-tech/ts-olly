# distroless static-debian12:nonroot - pinned for reproducible builds
# Update digest periodically: docker pull gcr.io/distroless/static-debian12:nonroot
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/ts-olly /
CMD ["/ts-olly"]
