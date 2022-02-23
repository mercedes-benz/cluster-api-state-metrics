#!/bin/bash
# SPDX-License-Identifier: MIT

set -e

CHART_PATH="deploy/chart"

for file in config/rbac/*role.yaml
do  
  TEMPLATE_PATH="${CHART_PATH}/templates/$(basename ${file})"
  sed -e '/^.*metadata:.*/a \ \ labels:\n\ \ \ \ {{- include "cluster-api-state-metrics.labels" . | nindent 4 }}' \
    -e 's/^  name:[[:space:]]\+\(.\+\)$/  name: {{ include "cluster-api-state-metrics.fullname" . }}-\1/' ${file} > ${TEMPLATE_PATH}
done

git diff --exit-code deploy/chart/templates/  || (echo "Changes to the helm chart templates have been detected! Make sure to update the chart version and publish a new release.")