#!/usr/bin/env bash
# Builds the h3xdOS ISO. Runs as root inside the h3xdos-build WSL distro.
set -euo pipefail

readonly project_dir=/mnt/a/h3xdOS
readonly profile_src=$project_dir/src/profile
readonly host_out_dir=$project_dir/out
readonly build_dir=/var/tmp/h3xdos
readonly stage_dir=$build_dir/profile
readonly work_dir=$build_dir/work
readonly out_dir=$build_dir/out
readonly unit_dir=etc/systemd/system

if (( EUID != 0 )); then
    echo "build.sh: must run as root" >&2
    exit 1
fi

stage_link() {
    local path=$1 target=$2
    mkdir -p -- "$stage_dir/airootfs/${path%/*}"
    ln -sfn -- "$target" "$stage_dir/airootfs/$path"
}

mkdir -p -- "$stage_dir" "$out_dir" "$host_out_dir"

# DrvFs reports every file as 0777 and mkarchiso keeps the mode bits it copies,
# so normalize here. Executables get their mode from file_permissions.
rsync -rtp --delete --chmod=D755,F644 -- "$profile_src/" "$stage_dir/"

# grep exits 1 when nothing matches; any other failure aborts the build.
crlf_files=$(grep -rlI $'\r' -- "$stage_dir" || [[ $? == 1 ]])
if [[ -n $crlf_files ]]; then
    printf 'build.sh: CRLF line endings in:\n%s\n' "$crlf_files" >&2
    exit 1
fi

unit_links=()
while read -r unit link; do
    [[ -z $unit || $unit == \#* ]] && continue
    if [[ -z $link ]]; then
        echo "build.sh: services.enable: no link given for $unit" >&2
        exit 1
    fi
    if [[ -e $stage_dir/airootfs/$unit_dir/$unit ]]; then
        target=/$unit_dir/$unit
    else
        target=/usr/lib/systemd/system/$unit
    fi
    [[ $link == */ ]] && link+=$unit
    stage_link "$unit_dir/$link" "$target"
    unit_links+=("$unit_dir/$link")
done < "$stage_dir/services.enable"

while read -r path target; do
    [[ -z $path || $path == \#* ]] && continue
    stage_link "$path" "$target"
done < "$stage_dir/airootfs.symlinks"

rm -rf -- "$work_dir"
rm -f -- "$out_dir"/*.iso
mkarchiso -v -w "$work_dir" -o "$out_dir" "$stage_dir"

# A dangling enable link means the unit's package is missing from packages.x86_64.
airootfs=$work_dir/x86_64/airootfs
for link in "${unit_links[@]}"; do
    if [[ ! -e $airootfs$(readlink -- "$airootfs/$link") ]]; then
        echo "build.sh: $link points to a unit that no installed package provides" >&2
        exit 1
    fi
done

iso_path=$(find "$out_dir" -maxdepth 1 -name '*.iso' -print -quit)
iso_name=${iso_path##*/}
iso_sum=$(cd "$out_dir" && sha256sum -- "$iso_name")

cp -- "$iso_path" "$host_out_dir/"
sums_file=$host_out_dir/SHA256SUMS
touch -- "$sums_file"
{
    awk -v name="$iso_name" '$2 != name' "$sums_file"
    printf '%s\n' "$iso_sum"
} > "$sums_file.new"
mv -- "$sums_file.new" "$sums_file"
(cd "$host_out_dir" && printf '%s\n' "$iso_sum" | sha256sum -c -)

echo "build.sh: $host_out_dir/$iso_name"
