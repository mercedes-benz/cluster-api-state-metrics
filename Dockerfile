# SPDX-License-Identifier: MIT
FROM gcr.io/distroless/base@sha256:3275e0193422021d412891b791f183b82bba943015aff9b7056758b7dd023fb4

LABEL org.opencontainers.image.source = https://github.com/Daimler/cluster-api-state-metrics

COPY bin/cluster-api-state-metrics /

ENTRYPOINT ["/cluster-api-state-metrics"]
