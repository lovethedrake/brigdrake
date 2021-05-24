FROM brigadecore/go-tools:v0.1.0
ARG VERSION
ARG COMMIT
ENV CGO_ENABLED=0
WORKDIR /go/src/github.com/lovethedrake/canard
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY vendor/ vendor/
RUN go build \
  -o bin/canard \
  -ldflags "-w -X github.com/lovethedrake/canard/pkg/version.version=$VERSION -X github.com/lovethedrake/canard/pkg/version.commit=$COMMIT" \
  ./cmd/canard

FROM scratch
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /go/src/github.com/lovethedrake/canard/bin/ /canard/bin/
CMD ["/canard/bin/canard"]
