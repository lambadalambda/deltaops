FROM debian:bookworm-slim

ARG DELTAOPS_BINARY=bin/deltaops-linux-amd64

RUN apt-get update \
	&& DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

COPY ${DELTAOPS_BINARY} /usr/local/bin/deltaops

RUN useradd --system --home-dir /nonexistent --no-create-home deltaops \
	&& mkdir -p /state \
	&& chown deltaops:deltaops /state \
	&& chmod 0700 /state

USER deltaops

ENTRYPOINT ["/usr/local/bin/deltaops"]
CMD ["run", "--state-dir", "/state"]
