#!/usr/bin/env bash
# Installs a package on the headset connected with adb (in developer mode), starts it, and shows
# what the program and the OpenXR runtime write to the log. Ctrl-C stops the log, not the program.
#
#   tools/install.sh build/cube.apk
#   APK_ID=org.example.plant tools/install.sh build/plant.apk   # a package built with its own id
set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "usage: tools/install.sh <a package, like build/cube.apk>" >&2
	exit 2
fi
package="$1"
if [[ -z "$(adb devices | sed -n '2,$p' | grep -w device || true)" ]]; then
	echo "install: no headset: connect it with a cable, in developer mode, and allow the computer in it" >&2
	exit 1
fi
name="$(basename "$package" .apk)"
id="${APK_ID:-io.github.akrck02.metaquest.${name//-/_}}"

adb install -r "$package"
adb logcat -c
adb shell am start -n "$id/android.app.NativeActivity"
adb logcat -s Go OpenXR
