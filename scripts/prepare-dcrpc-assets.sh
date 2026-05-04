#!/usr/bin/env sh
set -eu

version="v2.49.0"
base_url="https://github.com/chatmail/core/releases/download/$version"
repo_root=$(CDPATH= cd "$(dirname "$0")/.." && pwd -P)
asset_dir="${DELTAOPS_DCRPC_ASSET_DIR:-$repo_root/internal/notify/dcrpc/assets}"

sha256_file() {
	if command -v shasum >/dev/null 2>&1; then
		line=$(shasum -a 256 "$1")
	elif command -v sha256sum >/dev/null 2>&1; then
		line=$(sha256sum "$1")
	else
		printf 'missing checksum tool: install shasum or sha256sum\n' >&2
		return 1
	fi
	printf '%s' "${line%% *}"
}

asset_for_target() {
	case "$1" in
		linux/amd64)
			filename="deltachat-rpc-server-x86_64-linux"
			sha256="28e10b40518f55fa8ce20edd119fa743dd29a22df372b58443ec53eb753cb50c"
			;;
		linux/arm64)
			filename="deltachat-rpc-server-aarch64-linux"
			sha256="33acdc048060fcd51bc585f2eefdaa2cf93cca9306440f45be8c5936024732cf"
			;;
		linux/386)
			filename="deltachat-rpc-server-i686-linux"
			sha256="6fe6831f0bcd84316dafa416883249aba623eb392b7795769d7b9f635dc069b6"
			;;
		darwin/arm64)
			filename="deltachat-rpc-server-aarch64-macos"
			sha256="3ea30551ddaa67c2691c1cfbf0087ad95b799c5192269aada232ca2569891789"
			;;
		darwin/amd64)
			filename="deltachat-rpc-server-x86_64-macos"
			sha256="a8885769dc24eacd605b32593332de138fc77d97550b709c330d4fd4479b48c9"
			;;
		*)
			printf 'unsupported helper target %s\n' "$1" >&2
			printf 'supported targets: linux/amd64 linux/arm64 linux/386 darwin/arm64 darwin/amd64\n' >&2
			return 2
			;;
	esac
}

usage() {
	printf 'usage: sh scripts/prepare-dcrpc-assets.sh <target|all> [target...]\n'
	printf 'supported targets: linux/amd64 linux/arm64 linux/386 darwin/arm64 darwin/amd64\n'
}

if [ "$#" -eq 0 ]; then
	usage >&2
	exit 2
fi

if [ "$#" -eq 1 ] && [ "$1" = "all" ]; then
	set -- linux/amd64 linux/arm64 linux/386 darwin/arm64 darwin/amd64
fi

mkdir -p "$asset_dir"
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/deltaops-dcrpc.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

for target in "$@"; do
	asset_for_target "$target"
	url="$base_url/$filename"
	tmp="$tmp_dir/$filename"
	rm -f "$tmp"
	printf 'downloading %s from %s\n' "$target" "$url"
	curl -fL --proto '=https' --tlsv1.2 -o "$tmp" "$url"
	actual=$(sha256_file "$tmp")
	if [ "$actual" != "$sha256" ]; then
		rm -f "$tmp"
		printf 'checksum mismatch for %s\n' "$filename" >&2
		printf 'expected %s\n' "$sha256" >&2
		printf 'actual   %s\n' "$actual" >&2
		exit 1
	fi
	chmod 0700 "$tmp"
	mv "$tmp" "$asset_dir/$filename"
	printf 'installed %s\n' "$asset_dir/$filename"
done
