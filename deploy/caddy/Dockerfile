# M14.18. Stock caddy:2 ships neither a DNS-01 provider module nor a rate-limiting handler — both
# are third-party Caddy modules, only obtainable by building a custom binary with xcaddy (Caddy has
# no runtime plugin loading).
FROM caddy:2-builder AS builder

RUN xcaddy build \
	--with github.com/caddy-dns/cloudflare \
	--with github.com/mholt/caddy-ratelimit

FROM caddy:2

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
