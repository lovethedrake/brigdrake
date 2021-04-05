FROM brigadecore/go-tools:v0.1.0
ARG VERSION
ARG COMMIT
ENV CGO_ENABLED=0
WORKDIR /go/src/github.com/lovethedrake/brigdrake
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY vendor/ vendor/
RUN go build \
  -o bin/brigdrake-worker \
  -ldflags "-w -X github.com/lovethedrake/brigdrake/pkg/version.version=$VERSION -X github.com/lovethedrake/brigdrake/pkg/version.commit=$COMMIT" \
  ./cmd/brigdrake-worker

FROM scratch
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /go/src/github.com/lovethedrake/brigdrake/bin/ /brigdrake/bin/
CMD ["/brigdrake/bin/brigdrake-worker"]
