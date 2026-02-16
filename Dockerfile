FROM golang:1.24-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /vimvault ./cmd/vimvault

FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends neovim git ca-certificates && \
    rm -rf /var/lib/apt/lists/*
RUN useradd -m -s /bin/bash vimvault
COPY --from=builder /vimvault /usr/local/bin/vimvault
RUN mkdir -p /vault && chown vimvault:vimvault /vault
VOLUME /vault
EXPOSE 2222
USER vimvault
WORKDIR /vault
ENTRYPOINT ["vimvault"]
CMD ["--serve", "--vault", "/vault", "--listen", ":2222"]
