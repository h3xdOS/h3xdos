#!/usr/bin/env bash
# Provisions the h3xdos-build distro: keyring, mirrors, build toolchain and the
# builder user. Runs as root; bootstrap.ps1 calls it after installing the distro.
set -euo pipefail

readonly toolchain=(archiso git base-devel rsync reflector shellcheck namcap qemu-base edk2-ovmf)
# Approved by the user on 2026-10-23 so boot tests and the Hyprland check can show a window through WSLg.
# virtio-vga-gl loads the virtio-gpu, virtio-vga and virtio-gpu-gl modules, which pacman does not pull in.
readonly viewer_packages=(qemu-ui-gtk qemu-ui-opengl qemu-hw-display-virtio-vga-gl
    qemu-hw-display-virtio-gpu qemu-hw-display-virtio-vga qemu-hw-display-virtio-gpu-gl)
readonly mirror_countries=Czechia,Germany,Austria,Slovakia,Poland
readonly mirror_count=10
readonly builder_sudoers=/etc/sudoers.d/10-builder-pacman

if (( EUID != 0 )); then
    echo "provision.sh: must run as root" >&2
    exit 1
fi

# wsl --install --no-launch skips the image's first-launch script, which is
# what normally initializes the pacman keyring.
pacman-key --init
pacman-key --populate archlinux

pacman -Syu --noconfirm --needed reflector
reflector --country "$mirror_countries" --protocol https --age 12 --latest 20 \
    --sort rate --number "$mirror_count" --save /etc/pacman.d/mirrorlist
pacman -Syu --noconfirm --needed "${toolchain[@]}" "${viewer_packages[@]}"

# makepkg refuses to run as root, and makepkg -s needs pacman for dependencies.
id builder &>/dev/null || useradd -m -s /bin/bash builder
# sudo skips files in sudoers.d whose name contains a dot, so the candidate
# file stays inert until visudo accepts it.
printf '%s\n' 'builder ALL=(root) NOPASSWD: /usr/bin/pacman' > "$builder_sudoers.new"
chmod 0440 -- "$builder_sudoers.new"
visudo -cf "$builder_sudoers.new"
mv -- "$builder_sudoers.new" "$builder_sudoers"
