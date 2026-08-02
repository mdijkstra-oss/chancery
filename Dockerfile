# Chancery as a container: a Go builder, then a scratch image holding the static
# binary and the CA bundle and nothing else. Nothing outside this repository is
# read, so a git URL is a complete build context and a machine holding neither a
# clone nor a Go toolchain can still build the image.

FROM golang:1.26-alpine AS build

# The certificate bundle is installed in the builder because the final stage has
# no package manager. Without it an HTTPS backend fails at the handshake, which
# reads as a backend outage rather than a missing file.
RUN apk add --no-cache ca-certificates

WORKDIR /src

# The manifests are their own layer so that editing a source file does not
# refetch the module cache.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO off is what makes the binary static, which is what allows a scratch final
# stage. -s -w drop the symbol table and DWARF, -trimpath keeps build paths out
# of the binary; together they are most of the difference between an image a
# stranger builds in seconds and one they abandon.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /chancery ./cmd/chancery

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /chancery /chancery

# Nothing else is copied on purpose. No provider key, no configuration
# directory: the backend URL arrives as an environment variable and the Markdown
# is mounted read-only at run time, so the image is the same bytes for every
# deployment and editing a prompt costs a restart rather than a rebuild.

# Numeric, because scratch carries no /etc/passwd for a name to resolve against.
# The process reads its configuration and writes its log, and needs no identity
# beyond one that owns nothing.
USER 65532:65532

EXPOSE 8081

# The configuration path is named here rather than left to the caller because
# --config is required and has no default, so `docker run` alone would refuse.
# The flag stays overridable, and so does the subcommand.
ENTRYPOINT ["/chancery"]
CMD ["--config", "/config", "serve"]

# The binary checks itself, because a scratch image holds one file and no shell,
# no curl and no wget — any other command named here would be one the image
# cannot execute, leaving a container permanently unhealthy rather than one that
# is not yet listening. It asks only whether this process is serving; reaching a
# backend is deliberately not part of the answer, or a restart would follow every
# outage that was never chancery's.
HEALTHCHECK --interval=5s --timeout=3s --start-period=2s --retries=5 \
    CMD ["/chancery", "healthcheck", "--addr", "127.0.0.1:8081"]
