# Aktualizace H3XD

H3XD se aktualizuje přes internet, bez přeinstalace. Balíčky Arch Linuxu přicházejí
z oficiálních zrcadel, balíčky H3XD (vzhled, nastavení, nástroje) z repozitáře `[h3xdos]`
na GitHub Pages. Nastavení a soubory v domovské složce aktualizace nemění.

## Aktualizace systému

1. Otevřete menu H3XD a spusťte **System Update**, nebo v terminálu zadejte `h3xdos-update`.
2. Zadejte své heslo (sudo) a seznam balíčků potvrďte klávesou Enter.
3. Až okno napíše, že je potřeba restart, restartujte počítač.

System Update postupně:

| Krok | Co dělá |
|---|---|
| Arch Linux a H3XD | `sudo pacman -Syu`: všechny balíčky systému včetně jádra a balíčků H3XD |
| Drivers | `h3xdos-drivers` zjistí grafickou kartu a virtuální stroj a doinstaluje chybějící ovladače |
| H3XD applications | jednou nainstaluje výchozí aplikace (`h3xdos-apps`), pokud chybí |
| Kontejnery | aktualizuje kontejnery Ubuntu a Kali (`apt`, `kali-apt`), pokud existují |
| H3XD releases | ukáže nejnovější vydání ISO na GitHubu |

Na konci vypíše, jestli je potřeba restart: po aktualizaci jádra nebo prostředí H3XD.
Odhlášení nestačí, protože služba nastavení Xfce (xfconfd) po odhlášení běží dál.

Systém sám kontroluje aktualizace 10 minut po přihlášení a pak jednou denně. Když nějaké
čekají, ukáže upozornění. Jen kontrola bez instalace: `h3xdos-update --check`.

### První aktualizace z verze 1.0

Verze 1.0 má starší System Update, který zná jen první krok. Postup:

1. Spusťte System Update. Nainstaluje novou verzi updateru, nový vzhled a podporu hardwaru.
2. Spusťte System Update znovu. Teprve nová verze nainstaluje ovladače a aplikace.
3. Restartujte počítač.

V terminálu totéž: `h3xdos-update && h3xdos-update`, potom `sudo reboot`.

## Co přinesla aktualizace z 23. 10. 2026

- **Plocha:** popisky ikon na ploše jsou světlé (dříve tmavé na tmavé tapetě).
  Přepínač ploch v panelu má šestiúhelníky podle návrhu: aktivní plocha modrá, ostatní obrys.
- **Ovladače:** automatická detekce grafiky a virtuálního stroje, viz níže.
- **Výkon na slabších PC:**
  - swap v komprimované RAM (zram, až polovina paměti, nejvýš 4 GiB), rychlejší než swap na disku;
  - nastavení jádra pro zram (`vm.swappiness = 180` a další podle Arch Wiki);
  - týdenní TRIM pro SSD (`fstrim.timer`);
  - přehrávač mpv dekóduje video na grafické kartě (`hwdec=auto-safe`);
  - ovladače VA-API pro hardwarové dekódování videa u Intelu a AMD.
- **Aplikace:**
  - Chromium (prohlížeč);
  - Code - OSS (Visual Studio Code);
  - Mousepad (textový editor);
  - Ristretto (obrázky), Atril (PDF), mpv (video a hudba);
  - Xarchiver se 7-Zip, zip a unzip (archivy);
  - snímek obrazovky, správce úloh, kalkulačka;
  - fonty Noto s emoji;
  - GNOME Keyring pro hesla prohlížeče a editoru.
  
  Chromium je výchozí prohlížeč, Mousepad výchozí editor textu.

### Visual Studio Code

Balíček `code` je Code - OSS, otevřená verze Visual Studio Code z oficiálních repozitářů Archu.
Spouští se jako **Code - OSS** nebo příkazem `code`. Rozšíření bere z Open VSX místo
Microsoft Marketplace. Build od Microsoftu (`visual-studio-code-bin`) je jen v AUR
a jeho licence omezuje další šíření, proto ho H3XD neinstaluje.

## Ovladače

Bez parametrů `h3xdos-drivers` jen vypíše, co našel a co chybí:

    h3xdos-drivers
    sudo h3xdos-drivers --install

| Hardware | Balíčky |
|---|---|
| Intel | `vulkan-intel`, `intel-media-driver` (Broadwell a novější), `libva-intel-driver` (starší) |
| AMD | `vulkan-radeon` (dekódování videa je přímo v `mesa`) |
| NVIDIA Turing a novější (GTX 16xx, RTX) | `nvidia-open`, `nvidia-utils`, `nvidia-settings` |
| NVIDIA starší (GTX 10xx a starší) | `vulkan-nouveau`, zůstává ovladač nouveau |
| Intel nebo AMD spolu s NVIDIA (notebooky) | navíc `nvidia-prime`, program na kartě NVIDIA: `prime-run <program>` |
| KVM, QEMU, Proxmox | `qemu-guest-agent`, `spice-vdagent` |
| VirtualBox | `virtualbox-guest-utils` |
| VMware | `open-vm-tools` |
| Hyper-V | `hyperv` |

Po instalaci ovladače NVIDIA se znovu vytvoří initramfs a je potřeba restart. Starší karty
NVIDIA mají proprietární ovladač už jen v AUR, H3XD ho neinstaluje.

**Proxmox:** po aktualizaci zapněte ve webovém rozhraní u VM *Options → QEMU Guest Agent*
a VM vypněte a znovu zapněte (restart zevnitř nestačí). Proxmox pak umí VM čistě vypnout
a ukazuje její IP adresu.

## Vypnutí kroků

Soubor `/etc/h3xdos/update.conf`:

    install_drivers=yes
    install_default_apps=yes

Hodnota `no` krok vypne. Jednotlivé aplikace jde odinstalovat běžně, například
`sudo pacman -Rns galculator`. Updater je znovu neinstaluje, dokud zůstává nainstalovaný
balíček `h3xdos-apps`. Kdo nechce žádnou z nich, odinstaluje `h3xdos-apps`
a nastaví `install_default_apps=no`.

zram jde vypnout odkazem na `/dev/null`:

    sudo mkdir -p /etc/systemd/zram-generator.conf.d
    sudo ln -s /dev/null /etc/systemd/zram-generator.conf.d/50-h3xdos.conf

## Řešení problémů

- **`failed to synchronize all databases`.** Chybí připojení k internetu, nebo je zrcadlo
  nedostupné. Zkontrolujte síť a spusťte aktualizaci znovu.
- **Chyba podpisu nebo `invalid or corrupted package (PGP signature)`.** Zastaralá klíčenka:
  `sudo pacman -Sy archlinux-keyring`, potom znovu System Update.
- **`unable to lock database`.** Běží jiný pacman. Počkejte, až skončí. Soubor
  `/var/lib/pacman/db.lck` mažte jen tehdy, když žádný pacman neběží.
- **Instalace bez repozitáře `[h3xdos]`.** Starší testovací instalace ho nemusí mít v
  `/etc/pacman.conf`. Doplnění:

      printf '\n[h3xdos]\nSigLevel = Optional TrustedOnly\nServer = https://h3xdos.github.io/h3xdos-repo/$arch\n' | sudo tee -a /etc/pacman.conf

- **Nový vzhled se neprojevil.** Restartujte počítač, odhlášení nestačí.
- **Otázka `jack2` nebo `pipewire-jack`.** Vyberte `pipewire-jack`: posílá zvuk programů pro JACK
  přes PipeWire, který H3XD používá. Od `h3xdos-apps` 2026.10.23-2 se updater neptá. Kdo dříve
  zvolil `jack2`, dostane při další aktualizaci otázku `Remove jack2? [y/N]` a odpoví `y`.

## Pro správce: jak vydat aktualizaci

1. Upravte soubory v `src/pkgs/<balíček>/` a zvyšte v jeho `PKGBUILD` hodnotu `pkgrel`
   (u nové verze obsahu `pkgver` a `pkgrel=1`). Bez zvýšení verze pacman změnu nestáhne.
2. Postavte repozitář: `wsl.exe -d h3xdos-build -u root -- bash /mnt/a/h3xdOS/src/scripts/repo-build.sh`.
   Skript přestaví jen balíčky se změnou a skončí chybou, pokud namcap hlásí chybu.
3. Vyzkoušejte balíčky ve VM (`pacman -U` souborů z `A:\h3xdOS\repo`).
4. Publikujte: `bash /a/h3xdOS/src/scripts/repo-publish.sh` (Git Bash). GitHub Pages je
   nasadí během několika minut.
5. Zkontrolujte, že je verze venku: `pacman -Sy && pacman -Si <balíček>` na libovolném H3XD.
6. Zapište změnu do části „Co přinesla aktualizace“ v tomto souboru a commitněte zdrojáky.

Nový balíček se na existující systémy dostane jen jako závislost balíčku, který už mají
(tak přišel `h3xdos-hardware` přes `h3xdos-update`), nebo krokem updateru (tak přichází
`h3xdos-apps`).
