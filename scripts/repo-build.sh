#!/usr/bin/env bash
# Builds every package under src/pkgs and publishes them as the local [h3xdos]
# pacman repo. Runs as root inside the h3xdos-build WSL distro; makepkg runs as builder.
set -euo pipefail
shopt -s nullglob

readonly project_dir=/mnt/a/h3xdOS
readonly pkgs_src=$project_dir/src/pkgs
readonly host_repo_dir=$project_dir/repo
readonly build_dir=/var/tmp/h3xdos
readonly pkgs_stage=$build_dir/pkgs
readonly pkg_out=$build_dir/pkgout
readonly makepkg_build=$build_dir/makepkg
readonly srcdest=$build_dir/srcdest
readonly repo_dir=$build_dir/repo
readonly repo_name=h3xdos
readonly builder=builder

if (( EUID != 0 )); then
    echo "repo-build.sh: must run as root" >&2
    exit 1
fi

as_builder() {
    sudo -u "$builder" env PKGDEST="$pkg_out" BUILDDIR="$makepkg_build" SRCDEST="$srcdest" "$@"
}

install -d -o "$builder" -g "$builder" -- "$pkgs_stage" "$pkg_out" "$makepkg_build" "$srcdest"
# DrvFs reports every file as 0777; makepkg and namcap must see real modes.
rsync -rtp --delete --chmod=D755,F644 -- "$pkgs_src/" "$pkgs_stage/"
chown -R -- "$builder:$builder" "$pkgs_stage"

# grep exits 1 when nothing matches; any other failure aborts the build.
crlf_files=$(grep -rlI $'\r' -- "$pkgs_stage" || [[ $? == 1 ]])
if [[ -n $crlf_files ]]; then
    printf 'repo-build.sh: CRLF line endings in:\n%s\n' "$crlf_files" >&2
    exit 1
fi

# makepkg checks validpgpkeys against the builder keyring only.
for key in "$pkgs_stage"/*/keys/pgp/*.asc; do
    as_builder gpg --batch --quiet --import "$key"
done

built=()
for dir in "$pkgs_stage"/*/; do
    mapfile -t pkg_files < <(cd "$dir" && as_builder makepkg --packagelist)
    stale=0
    for pkg_file in "${pkg_files[@]}"; do
        if [[ ! -e $pkg_file || -n $(find "$dir" -newer "$pkg_file" -print -quit) ]]; then
            stale=1
        fi
    done
    if (( stale )); then
        # makepkg --syncdeps cannot resolve dependencies on our own packages, which
        # nothing compiles against, so install only what the Arch repos provide.
        mapfile -t deps < <(cd "$dir" && as_builder makepkg --printsrcinfo \
            | awk -F' = ' '$1 ~ /^\t(make|check)?depends$/ { print $2 }')
        official_deps=()
        for dep in "${deps[@]}"; do
            if pacman -Si -- "${dep%%[<>=]*}" &>/dev/null; then
                official_deps+=("$dep")
            fi
        done
        if (( ${#official_deps[@]} )); then
            pacman -S --needed --noconfirm --asdeps -- "${official_deps[@]}"
        fi
        (cd "$dir" && as_builder makepkg --nodeps --noconfirm --cleanbuild --force)
    fi
    # Debug symbols stay out of the repo; no installed system pulls them.
    for pkg_file in "${pkg_files[@]}"; do
        [[ $pkg_file == *-debug-* ]] || built+=("$pkg_file")
    done
done

namcap_report=$(namcap "${built[@]}")
printf '%s\n' "$namcap_report"
if grep -q ' E: ' <<<"$namcap_report"; then
    echo "repo-build.sh: namcap reported errors" >&2
    exit 1
fi

rm -rf -- "$repo_dir"
install -d -- "$repo_dir"
cp -- "${built[@]}" "$repo_dir/"
repo-add --quiet "$repo_dir/$repo_name.db.tar.gz" "$repo_dir"/*.pkg.tar.zst

# NTFS cannot hold the .db and .files symlinks repo-add creates, and a static
# host such as GitHub Pages needs them as plain files anyway.
install -d -- "$host_repo_dir"
rsync -rtL --delete -- "$repo_dir/" "$host_repo_dir/"

echo "repo-build.sh: ${#built[@]} packages in $host_repo_dir"
