# shellcheck shell=sh
# Commands exported from the Ubuntu and Kali containers land in ~/.local/bin.
# Appended, so host commands keep priority over exported ones.
case ":$PATH:" in
    *":$HOME/.local/bin:"*) ;;
    *) PATH="$PATH:$HOME/.local/bin" ;;
esac
