#!/usr/bin/env bash
# Publishes the [h3xdos] pacman repo to GitHub Pages (h3xdOS/h3xdos-repo).
# Runs in Git Bash on the Windows host, where git uses the gh credentials;
# repo-build.sh must have filled A:\h3xdOS\repo first.
set -euo pipefail

readonly repo_mirror=/a/h3xdOS/repo
readonly pages_checkout=/a/h3xdOS/publish/h3xdos-repo
readonly pages_remote=https://github.com/h3xdOS/h3xdos-repo.git
readonly arch=x86_64

if [[ ! -f $repo_mirror/h3xdos.db ]]; then
    echo "repo-publish.sh: $repo_mirror/h3xdos.db missing; run repo-build.sh first" >&2
    exit 1
fi

if [[ ! -d $pages_checkout/.git ]]; then
    git clone --quiet "$pages_remote" "$pages_checkout"
fi
cd "$pages_checkout"
git checkout --quiet -B main

rm -rf -- "${arch:?}"
mkdir -p -- "$arch"
cp -- "$repo_mirror"/* "$arch/"
# Pages would otherwise run Jekyll over the repo and could drop files.
touch .nojekyll

git add -A
if git diff --cached --quiet; then
    echo "repo-publish.sh: repo already up to date"
    exit 0
fi

packages=()
for pkg in "$arch"/*.pkg.tar.zst; do
    pkg=${pkg##*/}
    packages+=("${pkg%.pkg.tar.zst}")
done
git commit --quiet -F - <<EOF
Publish [h3xdos] repo

Packages: ${packages[*]}

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git push --quiet origin main
echo "repo-publish.sh: pushed $(git rev-parse --short HEAD) to $pages_remote"
