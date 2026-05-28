#!/usr/bin/env bash
set -euo pipefail

# generate-homebrew-formula.sh
# Generates a Homebrew formula for x-harness GitHub Release binaries.
# Usage: ./scripts/generate-homebrew-formula.sh <version> <checksums.txt>
# Example: ./scripts/generate-homebrew-formula.sh v0.99.0-rc1 .x-harness/release/go-binaries/checksums.txt

if [ $# -lt 2 ]; then
  echo "Usage: $0 <version> <checksums.txt>" >&2
  exit 1
fi

VERSION="${1}"
CHECKSUMS_FILE="${2}"

if [ ! -f "$CHECKSUMS_FILE" ]; then
  echo "Checksums file not found: $CHECKSUMS_FILE" >&2
  exit 1
fi

# Strip leading 'v' for formula version display if present
FORMULA_VERSION="${VERSION#v}"

# Read checksums into variables by parsing the sha256sum output.
# Expected filenames: x-harness-<version>-darwin-amd64, x-harness-<version>-darwin-arm64, etc.
DARWIN_AMD64_SHA=""
DARWIN_ARM64_SHA=""
LINUX_AMD64_SHA=""
LINUX_ARM64_SHA=""

while IFS=' ' read -r sha filename; do
  # sha256sum output: "<sha>  <filename>" (text) or "<sha> *<filename>" (binary)
  # Normalize: strip leading * and ./ before matching
  base="$(basename "$filename")"
  base="${base#\*}"
  case "$base" in
    x-harness-"${VERSION}"-darwin-amd64)
      DARWIN_AMD64_SHA="$sha"
      ;;
    x-harness-"${VERSION}"-darwin-arm64)
      DARWIN_ARM64_SHA="$sha"
      ;;
    x-harness-"${VERSION}"-linux-amd64)
      LINUX_AMD64_SHA="$sha"
      ;;
    x-harness-"${VERSION}"-linux-arm64)
      LINUX_ARM64_SHA="$sha"
      ;;
  esac
done < "$CHECKSUMS_FILE"

# Warn about missing checksums
missing=()
[ -z "$DARWIN_AMD64_SHA" ] && missing+=("darwin-amd64")
[ -z "$DARWIN_ARM64_SHA" ] && missing+=("darwin-arm64")
[ -z "$LINUX_AMD64_SHA" ] && missing+=("linux-amd64")
[ -z "$LINUX_ARM64_SHA" ] && missing+=("linux-arm64")

if [ ${#missing[@]} -gt 0 ]; then
  echo "Error: missing checksums for: ${missing[*]}" >&2
  exit 1
fi

cat <<EOF
class XHarness < Formula
  desc "Lightweight, verify-gated harness for AI-agent workflows"
  homepage "https://github.com/BrianNguyen29/x-harness"
  version "${FORMULA_VERSION}"
  license "MIT"

EOF

print_url_sha() {
  local os_arch="$1"
  local sha="$2"
  local goos="${os_arch%%-*}"
  local goarch="${os_arch##*-}"
  local hb_arch=""
  if [ "$goarch" = "amd64" ]; then
    hb_arch="intel"
  elif [ "$goarch" = "arm64" ]; then
    hb_arch="arm"
  fi

  local hb_os=""
  if [ "$goos" = "darwin" ]; then
    hb_os="macos"
  elif [ "$goos" = "linux" ]; then
    hb_os="linux"
  fi

  if [ -n "$sha" ]; then
    cat <<EOF
  on_${hb_os} do
    on_${hb_arch} do
      url "https://github.com/BrianNguyen29/x-harness/releases/download/${VERSION}/x-harness-${VERSION}-${goos}-${goarch}"
      sha256 "${sha}"
    end
  end

EOF
  fi
}

print_url_sha "darwin-amd64" "$DARWIN_AMD64_SHA"
print_url_sha "darwin-arm64" "$DARWIN_ARM64_SHA"
print_url_sha "linux-amd64" "$LINUX_AMD64_SHA"
print_url_sha "linux-arm64" "$LINUX_ARM64_SHA"

cat <<'EOF'
  def install
    bin.install Dir["x-harness-*"].first => "x-harness"
  end

  test do
    system "#{bin}/x-harness", "--version"
  end
end
EOF
