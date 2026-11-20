#!/bin/sh
# Installs the ebi-x binary from a GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/n-seiji/henge/main/install.sh | sh
#
# EBIX_VERSION (a tag, default latest) and EBIX_BIN_DIR (default ~/.local/bin)
# change what is installed and where.
set -eu

repo="n-seiji/henge"
version="${EBIX_VERSION:-latest}"
bin_dir="${EBIX_BIN_DIR:-$HOME/.local/bin}"

fail() {
	echo "install.sh: $*" >&2
	exit 1
}

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
linux | darwin) ;;
*) fail "unsupported OS: $os (build from source: go build ./cmd/ebi-x)" ;;
esac
case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) fail "unsupported CPU: $(uname -m)" ;;
esac

asset="ebi-x_${os}_${arch}.tar.gz"
if [ "$version" = latest ]; then
	base="https://github.com/$repo/releases/latest/download"
else
	base="https://github.com/$repo/releases/download/$version"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
curl -fsSL -o "$tmp/$asset" "$base/$asset" || fail "download $base/$asset"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt" || fail "download $base/checksums.txt"

grep "  $asset\$" "$tmp/checksums.txt" >"$tmp/expected" || fail "$asset is not in checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
	(cd "$tmp" && sha256sum -c expected >/dev/null) || fail "checksum mismatch for $asset"
else
	(cd "$tmp" && shasum -a 256 -c expected >/dev/null) || fail "checksum mismatch for $asset"
fi

tar -xzf "$tmp/$asset" -C "$tmp" ebi-x
if [ "$os" = darwin ]; then
	# Signed here rather than in the release so that no macOS runner is needed.
	# The hardened runtime keeps the commands Codex runs from attaching to
	# ebi-x and reading the keys it holds in memory.
	codesign --force --sign - --options runtime "$tmp/ebi-x"
fi

mkdir -p "$bin_dir"
# Renamed into place within bin_dir, not copied over the old file: writing
# into the binary of a running ebi-x fails on Linux, and the running one must
# keep its binary until `kill -HUP` re-executes the new one.
cp "$tmp/ebi-x" "$bin_dir/.ebi-x.new"
chmod 0755 "$bin_dir/.ebi-x.new"
mv -f "$bin_dir/.ebi-x.new" "$bin_dir/ebi-x"
echo "installed $bin_dir/ebi-x"

case ":$PATH:" in
*":$bin_dir:"*) ;;
*) echo "add $bin_dir to PATH to run ebi-x by name" ;;
esac
command -v codex >/dev/null 2>&1 || echo "Codex CLI was not found; install it and run: codex login"

cat <<'EOF'

Create a directory for your agent and initialize it:
  mkdir ~/ebi-x && cd ~/ebi-x && ebi-x init
EOF
