Place release-specific deltachat-rpc-server helper binaries in this directory before building a one-file DeltaOps release.

The pinned helper release is v2.49.0 from https://github.com/chatmail/core/releases/tag/v2.49.0.

Supported asset names and SHA-256 checksums:

- linux/amd64: deltachat-rpc-server-x86_64-linux, 28e10b40518f55fa8ce20edd119fa743dd29a22df372b58443ec53eb753cb50c
- linux/arm64: deltachat-rpc-server-aarch64-linux, 33acdc048060fcd51bc585f2eefdaa2cf93cca9306440f45be8c5936024732cf
- linux/386: deltachat-rpc-server-i686-linux, 6fe6831f0bcd84316dafa416883249aba623eb392b7795769d7b9f635dc069b6
- darwin/arm64: deltachat-rpc-server-aarch64-macos, 3ea30551ddaa67c2691c1cfbf0087ad95b799c5192269aada232ca2569891789
- darwin/amd64: deltachat-rpc-server-x86_64-macos, a8885769dc24eacd605b32593332de138fc77d97550b709c330d4fd4479b48c9

Prepare assets with:

```sh
sh scripts/prepare-dcrpc-assets.sh linux/amd64
```

Prepare one helper target per release build. `sh scripts/prepare-dcrpc-assets.sh all` is useful for local cache/testing, but the next build embeds every prepared helper.

The source tree intentionally does not commit these large binaries by default.
