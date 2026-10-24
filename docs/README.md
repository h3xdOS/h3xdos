# H3XD Operating System

H3XD je distribuce postavená na Arch Linuxu. Z live média se spouští jen grafický
instalátor, nainstalovaný systém má desktop Xfce 4.20 ve vlastním vzhledu H3XD
(téma `h3xd-dark`, rozpůlený hexagon jako jediný výrazný prvek, akcent `#1793D1`).

- Instalátor Calamares: GRUB pro UEFI i BIOS, ext4 nebo btrfs, volitelně LUKS,
  stránka Možnosti pro root účet, SSH server a root login přes SSH.
- Aktualizace: `pacman` z Arch repozitářů a z repozitáře `[h3xdos]` na GitHub Pages,
  nástroj **System Update** (`h3xdos-update`) a denní upozornění. Návod pro uživatele
  i správce: [AKTUALIZACE.md](AKTUALIZACE.md).
- Ovladače podle detekované grafiky (Intel, AMD, NVIDIA) a virtuálního stroje, zram swap
  a TRIM pro slabší PC, výchozí aplikace (Chromium, Code - OSS, Mousepad a další).
- Ovládání ve stylu Hyprlandu nad Xfce (Super+T, Super+1 až 5, dokování oken), vlastní
  vyhledávání aplikací, výběr Wi-Fi, kalendář a přihlašovací i zamykací obrazovka podle
  návrhu v2. Přehled zkratek je v [AKTUALIZACE.md](AKTUALIZACE.md).
- `apt` a `kali-apt` spouští skutečný apt v kontejnerech Ubuntu 24.04 a Kali Rolling
  (distrobox), Arch systém zůstává čistý.

Zdrojáky: <https://github.com/h3xdOS/h3xdos>, balíčky: <https://github.com/h3xdOS/h3xdos-repo>.

## Požadavky na build

- Windows 10 nebo 11 x64, WSL 2.
- Disk `A:` s NTFS a aspoň 60 GB volného místa. Všechno leží v `A:\h3xdOS`.
- Git for Windows a GitHub CLI (`gh`) přihlášené k organizaci h3xdOS, pokud se publikuje.

## Build od nuly

1. Ve zvýšeném PowerShellu jednou: `wsl --install --no-distribution`, `wsl --set-default-version 2`, restart.
2. Naklonovat zdrojáky do `A:\h3xdOS\src`.
3. V běžném PowerShellu: `powershell -NoProfile -File A:\h3xdOS\src\scripts\bootstrap.ps1`.
   Skript nainstaluje distro `h3xdos-build` na `A:\h3xdOS\wsl\h3xdos-build` a spustí
   `provision.sh` (klíče pacmanu, mirrory, archiso a nástroje, uživatel `builder`).
   Elevaci si nikdy nevyžádá sám; když je potřeba, vypíše příkaz.

## Příkazy

Z Git Bash, každé volání `wsl.exe` s prefixem `MSYS_NO_PATHCONV=1 WSL_UTF8=1`:

    balíčky:     wsl.exe -d h3xdos-build -u root -- bash /mnt/a/h3xdOS/src/scripts/repo-build.sh
    publikace:   bash /a/h3xdOS/src/scripts/repo-publish.sh
    ISO:         wsl.exe -d h3xdos-build -u root -- bash /mnt/a/h3xdOS/src/scripts/build.sh
    boot test:   wsl.exe -d h3xdos-build -u root -- bash /mnt/a/h3xdOS/src/scripts/boot-test.sh /var/tmp/h3xdos/out/<iso>
    živý test:   wsl.exe -d h3xdos-build -u root --exec env BOOT_TEST_DISPLAY=gtk,zoom-to-fit=on bash /mnt/a/h3xdOS/src/scripts/boot-test.sh <iso>

Pořadí je vždy `repo-build.sh`, pak `repo-publish.sh`, pak `build.sh`: ISO si balíčky
H3XD bere z lokálního repa `/var/tmp/h3xdos/repo`. Hotové ISO a `SHA256SUMS` jsou v
`A:\h3xdOS\out`, sériové logy boot testu `boot-bios.log` a `boot-uefi.log` vedle nich.
Boot test projde, když sériová konzole v BIOS i UEFI ukáže `h3xd login:`.

## Struktura

| Cesta | Obsah |
|---|---|
| `profile/` | archiso profil (z releng), `services.enable` a `airootfs.symlinks` místo symlinků |
| `pkgs/calamares` | PKGBUILD z AUR s ověřením podpisu release (klíč v `keys/pgp/`) |
| `pkgs/h3xdos-branding` | os-release, ikony a logo H3XD, tapeta, GRUB téma, Plymouth, fastfetch |
| `pkgs/h3xd-theme` | GTK 3, xfwm4 a xfce4-notifyd téma `h3xd-dark` |
| `pkgs/h3xdos-xfce-config` | panel, klávesové zkratky, terminál, prompt (zsh + starship), LightDM |
| `pkgs/h3xdos-containers` | `apt`, `kali-apt`, `h3xdos-box`, manifest distroboxu |
| `pkgs/h3xdos-calamares-config` | konfigurace instalátoru, branding, live relace jen s instalátorem |
| `pkgs/h3xdos-update` | `h3xdos-update`, položka System Update, uživatelský časovač, `/etc/h3xdos/update.conf` |
| `pkgs/h3xdos-hardware` | `h3xdos-drivers` (detekce GPU a VM), zram swap, nastavení swapu, TRIM, vypnutí systemd-networkd |
| `pkgs/h3xd-shell` | `h3xd-launcher`, `h3xd-dock`, `h3xd-network`, `h3xd-calendar`, `h3xd-panel-background` (Python, GTK 3) |
| `pkgs/h3xd-greeter` | přihlašovací a zamykací obrazovka pro LightDM (Python, GTK 3), s light-lockerem |
| `pkgs/h3xdos-apps` | metabalíček výchozích aplikací, výchozí programy pro typy souborů |
| `scripts/` | `bootstrap.ps1`, `provision.sh`, `repo-build.sh`, `repo-publish.sh`, `build.sh`, `boot-test.sh` |

## Aktualizace přes GitHub

`repo-build.sh` postaví balíčky do `[h3xdos]`, `repo-publish.sh` je nahraje do
`h3xdOS/h3xdos-repo` (složka `x86_64/`) a GitHub Pages je servíruje na
`https://h3xdos.github.io/h3xdos-repo/$arch`. Nainstalovaný systém má tento repozitář
v `/etc/pacman.conf`, takže `sudo pacman -Syu` nebo **System Update** stáhnou nové verze
brandingu, nastavení a nástrojů H3XD. Balíčky zatím nejsou podepsané
(`SigLevel = Optional TrustedOnly`), přenos chrání HTTPS. Postup vydání aktualizace
a co která aktualizace přinesla popisuje [AKTUALIZACE.md](AKTUALIZACE.md).

## Řešení problémů

Každý bod se během vývoje skutečně stal.

- **Git Bash přepisuje cesty.** Argumenty jako `/mnt/a/...` se v `wsl.exe` změní na cesty
  Windows. Pomůže `MSYS_NO_PATHCONV=1`.
- **Nečitelný výstup wsl.exe.** Bez `WSL_UTF8=1` tiskne UTF-16.
- **Proměnné zmizí.** `wsl.exe -d X -- bash -c '...$x...'` nejdřív projde login shellem distra,
  který `$x` rozbalí na prázdno. Pro inline příkazy `wsl.exe -d X -u root --exec bash -c '...'`.
- **Build na /mnt.** DrvFs hlásí všechny soubory jako 0777 a neumí symlinky. `build.sh`
  profil kopíruje na ext4 do `/var/tmp/h3xdos`, módy normalizuje a symlinky vytváří ze
  `services.enable` a `airootfs.symlinks`.
- **CRLF.** Bash skripty a PKGBUILDy s CRLF tiše selžou. `build.sh` i `repo-build.sh`
  build odmítnou, pokud najdou znak CR.
- **ISO zamčené VirtualBoxem.** Běžící VM drží připojené ISO a `build.sh` ho pak nepřepíše.
  Testovací VM proto bootuje kopii v `A:\h3xdOS\vm\`.
- **UEFI boot test selže v menu.** Stisk klávesy ve viditelném okně QEMU zastaví odpočet
  systemd-boot. Během živého testu do okna neklikat.
- **QEMU přes WSLg bez GPU.** `virtio-vga-gl` s `gl=on` přes WSLg visí (chybí DRI3 a DMABUF).
  Grafiku je lepší testovat ve VirtualBoxu.
- **Klávesy ve VirtualBoxu.** Windows si nechá Alt+Shift (přepínání jazyka) a Win+D.
  Ve VirtualBoxu zapnout zachytávání klávesnice (Host klávesa, standardně pravý Ctrl).
- **Černá plocha.** xfdesktop přes gdk-pixbuf 2.44 s glycinem nenačte SVG tapetu,
  proto `h3xdos-branding` při buildu renderuje PNG.
- **distrobox a sudo.** distrobox pod sudo odmítne běžet. `apt` z `h3xdos-containers`
  se při `sudo apt` vrátí k volajícímu uživateli přes `runuser`.
- **Velikost ISO pro GitHub Release.** Jeden soubor v release smí mít nejvýš 2 GiB.

## Otevřené body

- Podepisování balíčků `[h3xdos]` a ISO.
- Ovladače a výchozí aplikace už během instalace, když je síť. Zatím je doinstaluje první
  System Update; do ISO se aplikace nevejdou kvůli limitu 2 GiB pro soubor na GitHub Release.
- Vlastní aplikace Nastavení podle návrhu (obrazovka 2d); zatím slouží Nastavení Xfce v tématu `h3xd-dark`.
- Mezery mezi okny a dlaždicové skládání z návrhu xfwm4 neumí.
