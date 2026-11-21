#!/usr/bin/env bash
# Build the npm packages from the release archives in ./dist (run `make dist` first).
#
#   scripts/npm.sh build 0.0.1     → npm/dist/<package>/
#   scripts/npm.sh publish 0.0.1   → npm publish, platform packages first
set -euo pipefail

cmd=${1:?usage: scripts/npm.sh build|publish <version>}
version=${2:?usage: scripts/npm.sh build|publish <version>}
root=$(cd "$(dirname "$0")/.." && pwd)
out=$root/npm/dist
repo='{"type": "git", "url": "git+https://github.com/GunjanPatel359/burrow-G.git"}'

# npm name → Go archive name
declare -A targets=(
  [linux-x64]=linux_amd64
  [linux-arm64]=linux_arm64
  [darwin-x64]=darwin_amd64
  [darwin-arm64]=darwin_arm64
)

build() {
  rm -rf "$out" && mkdir -p "$out"

  for target in "${!targets[@]}"; do
    os=${target%-*}
    cpu=${target#*-}
    archive=$root/dist/burrow-g_${version}_${targets[$target]}.tar.gz
    dir=$out/burrow-g-$target
    mkdir -p "$dir/bin"
    tar -xzf "$archive" -C "$dir/bin" --strip-components=1 "burrow-g_${version}_${targets[$target]}/burrow-g"
    chmod 755 "$dir/bin/burrow-g"
    cp "$root/LICENSE" "$dir/"
    cat > "$dir/package.json" <<EOF
{
  "name": "burrow-g-$target",
  "version": "$version",
  "description": "burrow-G binary for $os $cpu",
  "license": "MIT",
  "repository": $repo,
  "os": ["$os"],
  "cpu": ["$cpu"],
  "files": ["bin/burrow-g", "LICENSE"]
}
EOF
  done

  main=$out/burrow-g
  mkdir -p "$main"
  cp -r "$root/npm/burrow-g/bin" "$main/"
  cp "$root/README.md" "$root/LICENSE" "$main/"
  cat > "$main/package.json" <<EOF
{
  "name": "burrow-g",
  "version": "$version",
  "description": "Find and resume your AI coding agent sessions across every agent and project, from one terminal UI.",
  "keywords": ["claude-code", "codex", "opencode", "copilot", "qoder", "ai", "agent", "sessions", "tui", "cli"],
  "homepage": "https://github.com/GunjanPatel359/burrow-G#readme",
  "bugs": "https://github.com/GunjanPatel359/burrow-G/issues",
  "license": "MIT",
  "repository": $repo,
  "bin": { "burrow-g": "bin/burrow-g.js" },
  "files": ["bin/burrow-g.js", "README.md", "LICENSE"],
  "os": ["linux", "darwin"],
  "engines": { "node": ">=16" },
  "optionalDependencies": {
    "burrow-g-linux-x64": "$version",
    "burrow-g-linux-arm64": "$version",
    "burrow-g-darwin-x64": "$version",
    "burrow-g-darwin-arm64": "$version"
  }
}
EOF
  echo "built npm packages in $out"
}

publish() {
  for target in "${!targets[@]}"; do
    (cd "$out/burrow-g-$target" && npm publish --access public)
  done
  (cd "$out/burrow-g" && npm publish --access public)
}

case $cmd in
  build) build ;;
  publish) publish ;;
  *) echo "unknown command: $cmd" >&2; exit 1 ;;
esac
