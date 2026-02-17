FROM golang:1.25-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /kopr ./cmd/kopr

FROM debian:bookworm-slim

ARG NVIM_VERSION=0.10.4
ARG TARGETARCH

RUN apt-get update && \
    apt-get install -y --no-install-recommends git ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*

RUN case "${TARGETARCH}" in \
        amd64) NVIM_ARCH="linux-x86_64" ;; \
        arm64) NVIM_ARCH="linux-aarch64" ;; \
        *) echo "unsupported arch: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    curl -fsSL "https://github.com/neovim/neovim/releases/download/v${NVIM_VERSION}/nvim-${NVIM_ARCH}.tar.gz" \
        | tar xz -C /usr/local --strip-components=1

RUN useradd -m -s /bin/bash kopr
COPY --from=builder /kopr /usr/local/bin/kopr
RUN mkdir -p /vault && chown kopr:kopr /vault
RUN mkdir -p /home/kopr/.config/kopr && chown -R kopr:kopr /home/kopr/.config
VOLUME /vault
EXPOSE 2222
USER kopr
WORKDIR /vault
ENTRYPOINT ["kopr"]
CMD ["--serve", "--vault", "/vault", "--listen", ":2222"]
