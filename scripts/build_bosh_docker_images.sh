#!/usr/bin/env bash

docker build -f ./cmd/registry/Dockerfile -t indicatorprotocol/bosh-indicator-protocol-registry .
docker build -f ./cmd/registry_proxy/Dockerfile -t indicatorprotocol/bosh-indicator-protocol-registry-proxy .
docker build -f ./cmd/registry_agent/Dockerfile -t indicatorprotocol/bosh-indicator-protocol-registry-agent .
