#!/bin/sh
set -eu

repository="DasDarki/nwm"
install_dir="${NWM_INSTALL_DIR:-$HOME/.local/bin}"
version="${NWM_VERSION:-latest}"

info() {
	printf '\033[1;35m==>\033[0m %s\n' "$1"
}

fail() {
	printf '\033[1;31merror:\033[0m %s\n' "$1" >&2
	exit 1
}

download() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL --retry 3 -o "$2" "$1"
	elif command -v wget >/dev/null 2>&1; then
		wget -q -O "$2" "$1"
	else
		fail "curl or wget is required"
	fi
}

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		fail "sha256sum or shasum is required to verify the download"
	fi
}

[ "$(uname -s)" = "Linux" ] || fail "this installer supports Linux only, use install.ps1 on Windows"

case "$(uname -m)" in
	x86_64 | amd64) architecture="amd64" ;;
	aarch64 | arm64) architecture="arm64" ;;
	*) fail "unsupported architecture $(uname -m)" ;;
esac

asset="nwm_linux_${architecture}"
if [ "$version" = "latest" ]; then
	base_url="https://github.com/${repository}/releases/latest/download"
else
	case "$version" in
		v*) ;;
		*) version="v${version}" ;;
	esac
	base_url="https://github.com/${repository}/releases/download/${version}"
fi

temporary="$(mktemp -d)"
trap 'rm -rf "$temporary"' EXIT INT TERM

info "Downloading ${asset} (${version})"
download "${base_url}/${asset}" "${temporary}/nwm" || fail "download failed, does release ${version} exist?"
download "${base_url}/checksums.txt" "${temporary}/checksums.txt" || fail "could not download checksums.txt"

expected="$(awk -v name="$asset" '$2 == name { print $1 }' "${temporary}/checksums.txt")"
[ -n "$expected" ] || fail "checksums.txt has no entry for ${asset}"
[ "$(sha256 "${temporary}/nwm")" = "$expected" ] || fail "checksum mismatch, refusing to install"

mkdir -p "$install_dir"
chmod 755 "${temporary}/nwm"
mv -f "${temporary}/nwm" "${install_dir}/nwm"
info "Installed nwm to ${install_dir}/nwm"

add_line() {
	file="$1"
	line="$2"
	mkdir -p "$(dirname "$file")"
	if [ -f "$file" ] && grep -Fq "$install_dir" "$file"; then
		return
	fi
	printf '\n%s\n' "$line" >>"$file"
	info "Added ${install_dir} to PATH in ${file}"
}

case ":${PATH}:" in
	*":${install_dir}:"*)
		path_ready=1
		;;
	*)
		path_ready=0
		export_line="export PATH=\"${install_dir}:\$PATH\""
		add_line "$HOME/.profile" "$export_line"
		if [ -f "$HOME/.bashrc" ] || [ "${SHELL##*/}" = "bash" ]; then
			add_line "$HOME/.bashrc" "$export_line"
		fi
		if [ -f "${ZDOTDIR:-$HOME}/.zshrc" ] || [ "${SHELL##*/}" = "zsh" ]; then
			add_line "${ZDOTDIR:-$HOME}/.zshrc" "$export_line"
		fi
		if command -v fish >/dev/null 2>&1 || [ "${SHELL##*/}" = "fish" ]; then
			add_line "${XDG_CONFIG_HOME:-$HOME/.config}/fish/conf.d/nwm.fish" "fish_add_path \"${install_dir}\""
		fi
		;;
esac

"${install_dir}/nwm" version || true
if [ "$path_ready" -eq 1 ]; then
	info "Done, run: nwm --help"
else
	info "Done, open a new terminal (or run: export PATH=\"${install_dir}:\$PATH\") and then: nwm --help"
fi
