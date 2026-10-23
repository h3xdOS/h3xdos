#!/usr/bin/env bash
# shellcheck disable=SC2034

iso_name="h3xdos"
iso_label="H3XD_$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y%m)"
iso_publisher="H3XD <https://github.com/h3xdOS/h3xdos>"
iso_application="H3XD Live"
iso_version="$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y.%m.%d)"
install_dir="arch"
bootmodes=('bios.syslinux'
           'uefi.systemd-boot')
pacman_conf="pacman.conf"
airootfs_image_type="squashfs"
airootfs_image_tool_options=('-comp' 'xz' '-Xbcj' 'x86,arm64' '-b' '1M' '-Xdict-size' '1M')
bootstrap_tarball_compression=('zstd' '-c' '-T0' '--auto-threads=logical' '--long' '-19')
kernel_params_aarch64="clk_ignore_unused pd_ignore_unused arm64.nopauth"
# ttyS0 carries the login prompt that scripts/boot-test.sh waits for; tty0 stays last so it remains /dev/console.
# The default 256M writable layer cannot hold a distrobox image; tmpfs only takes RAM as it fills.
kernel_params_x86_64="console=ttyS0,115200 console=tty0 cow_spacesize=4G quiet splash"
# mkarchiso applies an entry recursively only when the path ends in "/". The
# h3x home needs no entry: mkarchiso creates it from /etc/skel and runs
# chown -R for every passwd user with a uid between 1000 and 59999.
file_permissions=(
  ["/etc/shadow"]="0:0:400"
  ["/etc/gshadow"]="0:0:400"
  ["/etc/sudoers.d"]="0:0:750"
  ["/etc/sudoers.d/10-h3x-live"]="0:0:440"
  ["/root"]="0:0:750"
  ["/root/.automated_script.sh"]="0:0:755"
  ["/root/.gnupg"]="0:0:700"
  ["/usr/local/bin/choose-mirror"]="0:0:755"
  ["/usr/local/bin/Installation_guide"]="0:0:755"
  ["/usr/local/bin/livecd-sound"]="0:0:755"
)
