#!/usr/bin/env sh
set -eu

mode="${1:-smoke}"
repo_root=$(CDPATH= cd "$(dirname "$0")/.." && pwd -P)
compose_file="$repo_root/test/e2e/compose.yaml"
env_file="$repo_root/test/e2e/.env"
compose_project="${DELTAOPS_E2E_PROJECT:-deltaops-e2e}"
smoke_timeout="${DELTAOPS_E2E_SMOKE_TIMEOUT:-60}"
cd "$repo_root"

usage() {
	printf 'usage: sh scripts/e2e-docker.sh [smoke|live|build|reset]\n'
}

detect_target() {
	case "$(uname -m)" in
		arm64|aarch64)
			printf 'linux/arm64'
			;;
		x86_64|amd64)
			printf 'linux/amd64'
			;;
		*)
			printf 'linux/amd64'
			;;
	esac
}

configure_target() {
	target="${DELTAOPS_E2E_TARGET:-$(detect_target)}"
	case "$target" in
		linux/amd64)
			goarch="amd64"
			binary="bin/deltaops-linux-amd64"
			platform="linux/amd64"
			;;
		linux/arm64)
			goarch="arm64"
			binary="bin/deltaops-linux-arm64"
			platform="linux/arm64"
			;;
		*)
			printf 'unsupported Docker E2E target %s\n' "$target" >&2
			printf 'supported targets: linux/amd64 linux/arm64\n' >&2
			exit 2
			;;
	esac
	export DELTAOPS_BINARY="$binary"
	export DELTAOPS_DOCKER_PLATFORM="$platform"
}

compose_for() {
	project="$1"
	shift
	docker compose -p "$project" -f "$compose_file" "$@"
}

compose_live() {
	if [ -f "$env_file" ]; then
		docker compose --env-file "$env_file" -p "$compose_project" -f "$compose_file" "$@"
		return
	fi
	docker compose -p "$compose_project" -f "$compose_file" "$@"
}

build_image() {
	project="$1"
	configure_target
	printf 'preparing RPC helper for %s\n' "$target"
	sh "$repo_root/scripts/prepare-dcrpc-assets.sh" "$target"
	printf 'building %s\n' "$binary"
	GOOS=linux GOARCH="$goarch" MISE_TRUSTED_CONFIG_PATHS="$repo_root" mise exec -- go build -o "$repo_root/$binary" ./cmd/deltaops
	printf 'building Docker image for %s\n' "$platform"
	compose_for "$project" build
}

run_smoke() {
	smoke_project="$compose_project-smoke"
	build_image "$smoke_project"
	cleanup_smoke() {
		compose_for "$smoke_project" down -v >/dev/null 2>&1 || true
	}
	trap cleanup_smoke EXIT HUP INT TERM
	container=$(DELTAOPS_DCACCOUNT_URL= docker compose -p "$smoke_project" -f "$compose_file" run --detach deltaops run --state-dir /state --dcaccount-url dcaccount:placeholder)
	elapsed=0
	running=true
	while [ "$elapsed" -lt "$smoke_timeout" ]; do
		running=$(docker inspect -f '{{.State.Running}}' "$container" 2>/dev/null || printf 'false')
		if [ "$running" = "false" ]; then
			break
		fi
		sleep 1
		elapsed=$((elapsed + 1))
	done
	output=$(docker logs "$container" 2>&1 || true)
	if [ "$running" != "false" ]; then
		printf '%s\n' "$output"
		printf 'smoke failed: container did not exit within %s seconds\n' "$smoke_timeout" >&2
		docker rm -f "$container" >/dev/null 2>&1 || true
		exit 1
	fi
	status=$(docker inspect -f '{{.State.ExitCode}}' "$container" 2>/dev/null || printf '125')
	docker rm "$container" >/dev/null 2>&1 || true
	printf '%s\n' "$output"
	case "$output" in
		*"Delta Chat RPC helper is not packaged"*)
			printf 'smoke failed: build did not embed the RPC helper\n' >&2
			exit 1
			;;
	esac
	case "$output" in
		*"configure Delta Chat transport from provisioning URL"*)
			printf 'smoke passed: embedded RPC helper reached account provisioning\n'
			return 0
			;;
	esac
	printf 'smoke failed with unexpected status %s\n' "$status" >&2
	exit 1
}

run_live() {
	if [ -z "${DELTAOPS_DCACCOUNT_URL:-}" ] && [ ! -f "$env_file" ]; then
		printf 'DELTAOPS_DCACCOUNT_URL or %s is required for live Docker E2E runs\n' "$env_file" >&2
		exit 2
	fi
	build_image "$compose_project"
	compose_live up deltaops
}

case "$mode" in
	smoke)
		run_smoke
		;;
	live)
		run_live
		;;
	build)
		build_image "$compose_project"
		;;
	reset)
		compose_for "$compose_project" down -v
		compose_for "$compose_project-smoke" down -v
		;;
	-h|--help|help)
		usage
		;;
	*)
		usage >&2
		exit 2
		;;
esac
