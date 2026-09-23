#!/usr/bin/env bash
# Isolated API-server tests. Does not use kubectl, kubeconfig, or an AWS account.
set -euo pipefail
repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version=1.36.2
case "$(uname -s)-$(uname -m)" in
  Darwin-arm64)
    platform=darwin-arm64
    expected=9278f9e5af556b2f1f2d139769c1f0d717c7b4426917fdebdba898bcb725a916e4910d6160886194ff9be9589ea7c5c32c2b8ae0754703874b7c8ba8ddfc41ce ;;
  Linux-x86_64)
    platform=linux-amd64
    expected=ea743186c8a799f5cf8faf16969f86189d003cb7d130e0ac4b58789f1e5748dcf30ebe91c837a10d5ac415383da3e10b9e64d65785c938c23e739781cfb76f08 ;;
  *) echo "Unsupported platform; set KUBEVISTA_TEST_ASSETS and run the Go test directly." >&2; exit 1 ;;
esac
assets="${repository_root}/work/envtest-${version}-${platform}"
archive="${assets}/envtest.tar.gz"
mkdir -p "$assets"
curl --fail --location --retry 3 --output "$archive" "https://github.com/kubernetes-sigs/controller-tools/releases/download/envtest-v${version}/envtest-v${version}-${platform}.tar.gz"
actual="$(shasum -a 512 "$archive" | awk '{print $1}')"
if [[ "$actual" != "$expected" ]]; then
  echo "envtest checksum mismatch; refusing to execute downloaded binaries" >&2
  exit 1
fi
tar -xzf "$archive" -C "$assets"
export KUBEVISTA_TEST_ASSETS="${assets}/controller-tools/envtest"
cd "${repository_root}/api"
go test -race ./internal/kubernetes -run TestDurableRealAPI -count=1 -timeout=180s -v
