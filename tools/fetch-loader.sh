#!/usr/bin/env bash
# Downloads the Khronos OpenXR loader for Android, the newest release on Maven Central, and keeps
# its arm64 library in third_party/openxr-loader/ (not committed), where build-apk.sh finds it.
#
#   tools/fetch-loader.sh            # the newest release
#   tools/fetch-loader.sh 1.1.49     # a version of your choosing
set -euo pipefail

repository="https://repo1.maven.org/maven2/org/khronos/openxr/openxr_loader_for_android"
root="$(cd "$(dirname "$0")/.." && pwd)"
destination="$root/third_party/openxr-loader"

version="${1:-}"
if [[ -z "$version" ]]; then
	# maven-metadata.xml names the newest release in <release>.
	version="$(curl --fail --silent --show-error "$repository/maven-metadata.xml" |
		sed -n 's:.*<release>\(.*\)</release>.*:\1:p')"
fi
if [[ -z "$version" ]]; then
	echo "fetch-loader: Maven Central named no release" >&2
	exit 1
fi

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
archive="$work/openxr_loader_for_android-$version.aar"
echo "fetch-loader: downloading openxr_loader_for_android $version"
curl --fail --silent --show-error --location -o "$archive" "$repository/$version/openxr_loader_for_android-$version.aar"

# An AAR is a zip: the library for each processor is under jni/. The headset is arm64.
# Its licence travels with it, for THIRD_PARTY_NOTICES.md.
unzip -q -o "$archive" "jni/arm64-v8a/libopenxr_loader.so" "META-INF/LICENSE" -d "$work/extracted"
mkdir -p "$destination/arm64-v8a"
cp "$work/extracted/jni/arm64-v8a/libopenxr_loader.so" "$destination/arm64-v8a/"
cp "$work/extracted/META-INF/LICENSE" "$destination/LICENSE"
echo "$version" >"$destination/VERSION"
echo "fetch-loader: $destination/arm64-v8a/libopenxr_loader.so ($version)"
