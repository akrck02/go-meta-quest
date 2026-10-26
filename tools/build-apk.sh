#!/usr/bin/env bash
# Builds a Go program into a package for the Meta Quest 3, signed with the debug key:
# build/<name>.apk, in the folder it is run from. It builds from the current folder, so it works for
# this repository's programs and for a module of your own alike.
#
#   tools/build-apk.sh ./cmd/cube
#   APK_ID=org.example.plant APK_LABEL="My plant" /path/to/go-meta-quest/tools/build-apk.sh ./cmd/plant
#
# The package's id is io.github.akrck02.metaquest.<name> and the name under its icon the folder's,
# capitalised, unless APK_ID and APK_LABEL say otherwise: a program of your own wants its own id.
#
# It needs the Android SDK (build tools and a platform) and the NDK. It finds them in ANDROID_HOME
# and ANDROID_NDK_HOME, or in Homebrew's android-commandlinetools; the OpenXR loader comes from
# tools/fetch-loader.sh, which it runs the first time.
set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "usage: tools/build-apk.sh <the program's folder, like ./cmd/cube>" >&2
	exit 2
fi
root="$(cd "$(dirname "$0")/.." && pwd)"
program="$1"
name="$(basename "$program")"

# The Android tools: the newest build tools, platform and NDK there are.
sdk="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-/opt/homebrew/share/android-commandlinetools}}"
newest() { ls -1 "$1" 2>/dev/null | sort -V | tail -n 1; }
ndk="${ANDROID_NDK_HOME:-$sdk/ndk/$(newest "$sdk/ndk")}"
buildTools="$sdk/build-tools/$(newest "$sdk/build-tools")"
platform="$sdk/platforms/$(newest "$sdk/platforms")"
compiler="$ndk/toolchains/llvm/prebuilt/$(uname -s | tr '[:upper:]' '[:lower:]')-x86_64/bin/aarch64-linux-android29-clang"
for needed in "$buildTools/aapt2" "$buildTools/zipalign" "$buildTools/apksigner" "$platform/android.jar" "$compiler"; do
	if [[ ! -e "$needed" ]]; then
		echo "build-apk: $needed is missing: set ANDROID_HOME to the SDK and ANDROID_NDK_HOME to the NDK" >&2
		exit 1
	fi
done

loader="$root/third_party/openxr-loader/arm64-v8a/libopenxr_loader.so"
if [[ ! -f "$loader" ]]; then
	"$root/tools/fetch-loader.sh"
fi

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/lib/arm64-v8a" build

# The program, as the shared library NativeActivity loads (libmain.so). 16 KB pages: Android 15
# and later may run with them, and a library aligned to 4 KB would not load there.
echo "build-apk: building $program"
CGO_ENABLED=1 GOOS=android GOARCH=arm64 CC="$compiler" \
	go build -buildmode=c-shared -trimpath -ldflags=-extldflags=-Wl,-z,max-page-size=16384 \
	-o "$work/lib/arm64-v8a/libmain.so" "$program"
rm -f "$work/lib/arm64-v8a/libmain.h" # cgo's header for the exports: the package needs none
cp "$loader" "$work/lib/arm64-v8a/"

# The manifest, with this program's id and name.
id="${APK_ID:-io.github.akrck02.metaquest.${name//-/_}}"
label="${APK_LABEL:-$(echo "${name:0:1}" | tr '[:lower:]' '[:upper:]')${name:1}}"
sed -e "s/@PACKAGE@/$id/g" -e "s/@LABEL@/$label/g" -e "s/@LIBRARY@/main/g" \
	"$root/tools/AndroidManifest.xml" >"$work/AndroidManifest.xml"

# The package: the manifest linked by aapt2, then the libraries stored uncompressed (the system
# maps them straight from the package), aligned to 16 KB pages, and signed.
"$buildTools/aapt2" link -o "$work/unaligned.apk" -I "$platform/android.jar" --manifest "$work/AndroidManifest.xml"
(cd "$work" && zip -q -0 -r unaligned.apk lib)
"$buildTools/zipalign" -f -P 16 4 "$work/unaligned.apk" "$work/aligned.apk"

keystore="$HOME/.android/debug.keystore"
if [[ ! -f "$keystore" ]]; then
	mkdir -p "$(dirname "$keystore")"
	keytool -genkeypair -keystore "$keystore" -storepass android -alias androiddebugkey -keypass android \
		-keyalg RSA -validity 10000 -dname "CN=Android Debug,O=Android,C=US"
fi
output="$PWD/build/$name.apk"
# the debug key's passwords are Android's well-known ones
"$buildTools/apksigner" sign --ks "$keystore" --ks-pass pass:android --ks-key-alias androiddebugkey \
	--key-pass pass:android --out "$output" "$work/aligned.apk"
rm -f "$output.idsig" # apksigner's v4 signature, for streaming installs only
echo "build-apk: $output"
