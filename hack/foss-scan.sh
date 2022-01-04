#!/bin/bash
# SPDX-License-Identifier: MIT

set -euo pipefail

START_TIME="$(date +%s)"
echo "$(date '+%H:%M:%S') foss: start"

function try_delete_version {
	local version_name=${1}
	local bearer_token=${2}
	local project_url=${3}

	echo "Trying to retrieve version with name ${version_name}"
	version=$(curl -k -s -X GET \
		-H 'Accept: application/vnd.blackducksoftware.project-detail-4+json' \
		-H "Authorization: Bearer ${bearer_token}" "${project_url}"/versions \
		| jq -r '.items[] | select(.versionName == "'"${version_name}"'") | ._meta.href')

	if [[ -n ${version} ]]; then
		echo "Trying to delete version \"${version}\""
		curl -k -s -X DELETE -H "Accept: application/vnd.blackducksoftware.project-detail-4+json" -H "Authorization: Bearer ${bearer_token}" "${version}"
	else
		echo "Version with name ${version_name} could not be retrieved. It might not have been created yet"
	fi
}

TARGET_FILE_NAME=${1:-""}
TARGET_FILE_NAME_NOTICES=${2:-""}

: "${BLACKDUCK_SCAN_VERSION_NAME?"must be set"}"

report_timeout=3600

if [[ ! "$(command -v go)" ]]; then
	echo "Expected go binary in PATH"
	exit 1
else
	echo "Possibly using different version of go than the one in WORKSPACE."
	echo "Using $(go version)"
fi

echo "get bearer token"
bearer_token=$(curl -k -s -X POST \
	-H "Accept: application/vnd.blackducksoftware.user-4+json" \
	-H "Authorization: token ${BLACKDUCK_TOKEN}" \
	"${BLACKDUCK_URL}"/api/tokens/authenticate \
	| jq -rc '.bearerToken')

echo "get project id (i.e. project url) with the project name"
project_url=$(curl -k -s -X GET \
	-H 'Accept: application/vnd.blackducksoftware.project-detail-4+json' \
	-H "Authorization: Bearer ${bearer_token}" \
	"${BLACKDUCK_URL}"/api/projects\?q\=name:"${BLACKDUCK_PROJECT_NAME}" \
	| jq -rc '.items[] | select(.name == "'"${BLACKDUCK_PROJECT_NAME}"'") | ._meta.href')

echo "get all versions"
versions=$(curl -k -s -X GET \
	-H 'Accept: application/vnd.blackducksoftware.project-detail-4+json' \
	-H "Authorization: Bearer ${bearer_token}" "${project_url}"/versions \
	| jq -rc '[.items[] | {name: .versionName, created: .createdAt, reportLink: ._meta.href }] | sort_by(.created) | reverse')

count=$(jq '. | length' <<< "${versions}")

# The number 8 for keep scans because of 6 possible concurrent runs and 2 as
# buffer for longer access.
keep_scans=8
echo "looping through all ${count} versions and delete all versions except the newest ${keep_scans}"
for i in $(seq 0 $((count - 1))); do
	if [[ $i -ge ${keep_scans} ]]; then
		version=$(jq -r ".[$i].reportLink" <<< "${versions}")
		echo "delete version $(jq ".[$i].name" <<< "${versions}")"
		curl -k -s -X DELETE -H "Accept: application/vnd.blackducksoftware.project-detail-4+json" -H "Authorization: Bearer ${bearer_token}" "${version}"
	fi
done

trap 'try_delete_version ${BLACKDUCK_SCAN_VERSION_NAME} ${bearer_token} ${project_url}' SIGINT SIGTERM

mkdir -p "bin/scan-result/"

echo "create bd scan ${BLACKDUCK_SCAN_VERSION_NAME}"
set +e
bash <(curl -s -L https://detect.synopsys.com/detect.sh) \
	--blackduck.url="${BLACKDUCK_URL}" \
	--blackduck.api.token="${BLACKDUCK_TOKEN}" \
	--detect.project.name="${BLACKDUCK_PROJECT_NAME}" \
	--detect.project.version.name="${BLACKDUCK_SCAN_VERSION_NAME}" \
	--detect.timeout="${report_timeout}" \
	--detect.policy.check.fail.on.severities="BLOCKER" \
	--blackduck.trust.cert=true \
	--blackduck.hub.auto.import.cert=true \
	--detect.cleanup=false \
	--detect.risk.report.pdf=true \
	--detect.risk.report.pdf.path=bin/scan-result/ \
	--detect.notices.report=true \
	--detect.notices.report.path=bin/scan-result/ \
	--insecure
RC=$?

# Delete the scan if it completed successfully.
if [[ ${RC} == 0 ]]; then
	try_delete_version "${BLACKDUCK_SCAN_VERSION_NAME}" "${bearer_token}" "${project_url}"
fi
set -e

mv bin/scan-result/*BlackDuck_RiskReport.pdf bin/scan-result/BlackDuck_RiskReport.pdf
mv bin/scan-result/*Black_Duck_Notices_Report.txt bin/scan-result/Black_Duck_Notices_Report.txt

exit $RC

echo "$(date '+%H:%M:%S') foss: end (Elapsed time: $(($(date +%s) - START_TIME))s)"
