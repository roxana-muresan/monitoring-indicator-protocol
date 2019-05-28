#!/bin/bash

set -efu

fly --target indipro set-pipeline \
    --pipeline indicator-protocol \
    --config "ci/pipeline.yml"
