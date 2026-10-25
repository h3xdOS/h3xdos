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
| Flatpak applications | `flatpak update` aplikací z Flathubu a úklid nepoužívaných runtime |
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

## Aktualizace 1.0.5 (25. 10. 2026)

Po aktualizaci restartujte počítač; klávesy hlasitosti nastaví až další přihlášení.

- **Volume** (v menu Multimedia, nebo Super a „vol", nebo kliknutím na `vol` v panelu):
  vlastní správce hlasitosti H3XD místo dřívějšího Volume Control (pavucontrol). Změny
  odjinud, například zapojená sluchátka, ukáže hned.
  - **Output:** výstup, kterým zvuk hraje (při více zařízeních výběr nahoře), hlasitost
    a Mute, měřič úrovně toho, co hraje, konektor (například reproduktory nebo
    sluchátka) a vyvážení vlevo a vpravo. Pod tím další výstupy, tlačítko Use z nich
    udělá hlavní. Přepínač **Louder than 100 %** pustí hlasitost až na 150 %.
  - **Input:** totéž pro mikrofony; měřič ukáže, jestli mikrofon slyší.
  - **Applications:** hlasitost každého programu, který hraje nebo nahrává, a kam hraje
    nebo odkud nahrává. Program nahrávající zvuk počítače si vybere „Monitor of …".
  - **Devices:** profil každé zvukové karty, například zvuk přes HDMI, jen výstup nebo
    vypnuto.
- **Klávesy hlasitosti** (hlasitost nahoru a dolů po 5 %, ztlumení, ztlumení mikrofonu)
  fungují a ukážou úroveň dole uprostřed obrazovky. Účtům, které si tyto klávesy
  přiřadily k něčemu jinému, zůstanou jejich zkratky.
- pavucontrol zůstává nainstalovaný, nové instalace ho nemají. Odebrání:
  `sudo pacman -Rns pavucontrol`.

## Aktualizace 1.0.4.1 (25. 10. 2026)

Vzhled ovládacích prvků. Po aktualizaci restartujte počítač; aplikace spuštěné před
aktualizací si nový vzhled načtou až po novém přihlášení.

- **Přepínač H3XD:** zaškrtávací políčka a přepínače mají tvar šestiúhelníkového
  přepínače z návrhu. Vypnuto je šedé s jezdcem vlevo, zapnuto modré s jezdcem vpravo,
  neurčitý stav (část položek vybraná) tmavě modrý s jezdcem uprostřed. Platí pro
  aplikace GTK 3 (nastavení Xfce, Thunar, aplikace H3XD), GTK 4 a libadwaita včetně
  aplikací z Flathubu a pro instalátor. V nabídkách a seznamech je přepínač menší.
- **Kabel v panelu:** při připojení kabelem ukazuje panel ikonu RJ45 z ikon H3XD a název
  síťového rozhraní (například `enp3s0` nebo `eth0`) místo dlouhého „Wired connection 1".
  Název spojení je v bublině po najetí myší, kliknutí otevře nastavení sítě.
- **Karty v dialozích** (například nastavení panelu) mají barvu okna místo šedé.

## Aktualizace 1.0.4 (25. 10. 2026)

- **Firewall** (v menu System, nebo Super a „fire"): vlastní aplikace H3XD nad ufw. Při
  otevření chce heslo; platí, dokud okno nezavřete.
  - **Overview:** firewall zapnout a vypnout, co projde bez pravidla, kolik je pravidel
    a které služby jsou povolené.
  - **Rules:** všechna pravidla ufw v pořadí, v jakém je ufw zkouší (první, které sedí,
    rozhoduje). Barevný štítek ALLOW, DENY, REJECT nebo LIMIT, směr IN nebo OUT, port,
    adresa a komentář. Přepínačem jde pravidlo vypnout a zase zapnout, Delete ho smaže.
    **Add rule:** akce, směr, port (`22`, `80,443`, `6000:6007`), protokol, odkud a
    komentář; v části Advanced cílová adresa, síťové rozhraní, logování a pozice.
  - **Services:** jedním přepínačem SSH, web server, KDE Connect, LocalSend, sdílení
    složek Windows, tiskárny, mDNS, Syncthing, Steam Remote Play, Minecraft server
    a vzdálená plocha. Pod tím programy, které na počítači čekají na spojení, a zda je
    firewall pustí (Allow přidá pravidlo).
  - **Activity:** spojení, která firewall zablokoval (z logu jádra), s tlačítkem Allow.
  - **Settings:** co se stane se spojením bez pravidla (příchozí, odchozí, směrované),
    úroveň logování a reset firewallu.
- Firewall se aktualizací nezapne sám. Po zapnutí v aplikaci startuje i s počítačem.
  Když běží SSH server, aplikace před zapnutím přidá pravidlo pro SSH, aby se vzdálený
  přístup nezavřel.
- ufw nezná vypnutá pravidla; H3XD si je ukládá v `/var/lib/h3xd-firewall/disabled.json`
  a při zapnutí je vrací (blokující pravidla na začátek, povolující na konec).

## Aktualizace 1.0.3.1 (25. 10. 2026)

Terminál podle návrhu. Po aktualizaci restartujte počítač; při přihlášení H3XD přenastaví
okno terminálu ve vašem účtu.

- **Uvítání:** nové okno nebo karta terminálu ukáže logo H3XD a přehled systému
  (fastfetch) s paletou barev. Vypnutí: `mkdir -p ~/.config/h3xd && touch ~/.config/h3xd/no-greeting`.
- **Prompt v blocích:** uživatel, složka a ve složce s gitem větev, na druhém řádku
  červené `>`. Vpravo délka příkazu nad 2 s, kód chyby a čas. Dřívější verze ukazovaly místo něj prompt grml
  (`h3x@h3xd ~ %`), který ten z návrhu před každým řádkem přepsal. Vlastní
  `~/.config/starship.toml` má přednost; dosavadní nezměněná kopie z dřívějších verzí se
  smaže, aby se prompt dál aktualizoval.
- **Barvy při psaní:** známý příkaz zeleně, neznámý červeně, přepínače žlutě, cesty modře;
  šedě návrh dokončení z historie (šipka vpravo ho přijme).
- **Barevné `ls`** (eza): složky modře s lomítkem, spustitelné soubory zeleně, práva a data
  tlumeně. `ll` dlouhý výpis, `la` i skryté soubory, `lt` strom. Původní `ls` je
  `command ls`.
- **pacman v barvách** (`Color` v `/etc/pacman.conf`).
- **Okno terminálu:** odsazení textu od okrajů, větší výchozí velikost 100x28, blikající
  blokový kurzor v modré, bez zvonku. Odsazení přichází z `~/.config/gtk-3.0/gtk.css`
  (import `/usr/share/h3xd/gtk-3.0/user.css`), protože terminál si jinak nastaví vlastní.

## Aktualizace 1.0.3 (25. 10. 2026)

Po aktualizaci restartujte počítač. Při prvním přihlášení H3XD upraví lištu a barvy
aplikací GTK 4 ve vašem účtu.

- **Obchod H3XD Apps** (v menu nebo Super a „apps"): vybrané aplikace z repozitářů Archu
  a celý Flathub v jednom okně. Vlevo Discover, Installed a kategorie, nahoře hledání.
  Tlačítko Install, průběh instalace v kartě aplikace, pak Open. V části Installed jde
  aplikace odebrat.
  - Aplikace z Archu chtějí heslo. Instalace zároveň aktualizuje celý systém (Arch nesnese
    instalaci ze starších seznamů balíčků), takže může trvat déle; když se aktualizuje
    jádro, karta řekne, že je potřeba restart.
  - Aplikace z Flathubu se instalují bez hesla. Steam, Spotify, Heroic a další, které
    v repozitářích Archu nejsou, jsou tady.
  - Discord: obchod po instalaci rovnou stáhne samotný Discord (přes 600 MB) a ukáže
    průběh. Z menu se pak spouští hned.
- **Aplikace na pozadí:** ikony v liště (Discord a další) jsou v jedné řadě uvnitř lišty.
  Kliknutí na ikonu otevře menu aplikace (např. Open Discord). Zavřené okno Discordu
  dál běží v liště.
- **Aplikace GTK 4** (libadwaita, např. dialogy zenity) jsou tmavé a v barvách H3XD.
  Menu ikon v liště mají písmo IBM Plex Sans jako ostatní menu.
- **System Update** aktualizuje i aplikace z Flathubu. Vypnutí: `update_flatpaks=no`
  v `/etc/h3xdos/update.conf`.

### Proč nejde nainstalovat .deb

Soubory `.deb` jsou balíčky pro Debian a Ubuntu, H3XD stojí na Arch Linuxu. Aplikace
hledejte v H3XD Apps (repozitáře Archu a Flathub). Program, který existuje jen jako `.deb`,
poběží v kontejneru Ubuntu (`apt`).

Aplikaci spuštěnou z terminálu ukončí zavření terminálu. Spouštějte ji z menu nebo ze
spouštěče (Super), pak běží samostatně.

## Aktualizace 1.0.2.1 (25. 10. 2026)

Po aktualizaci restartujte počítač. Při prvním přihlášení H3XD přepne účet na nové ikony.

- **Ikony H3XD:** vlastní sada ikon v šestiúhelníku a v barvách návrhu, výchozí motiv
  ikon systému místo Papirus-Dark. Kreslí 140 ikon, které H3XD ukazuje: menu
  a jeho kategorie, aplikace, složky a místa na ploše a v Thunaru, disky, typy souborů,
  stav (hlasitost, baterie, Wi-Fi), dialogy, odhlášení a vypnutí.
- Ikony, které sada zatím nemá, a symbolické ikony v nástrojových lištách bere dál
  Papirus-Dark. Logo H3XD zůstává ze sady loga ve všech velikostech.
- Kdo si v **Settings > Appearance > Icons** vybral jiný motiv, tomu aktualizace ikony
  nezmění. Zpět na nové ikony: vyberte tam **H3XD**.

## Aktualizace 1.0.2 (24. 10. 2026)

Po aktualizaci restartujte počítač. Při prvním přihlášení H3XD jednou upraví nastavení
účtu: nový panel, klávesové zkratky, pět ploch a zámek obrazovky. Vlastní úpravy panelu
se přitom přepíšou, ostatní vlastní zkratky zůstanou.

- **Klávesové zkratky ve stylu Hyprlandu:**

  | Zkratka | Co dělá |
  |---|---|
  | Super, Super+F, Super+Enter | vyhledávání aplikací |
  | Super+T | terminál |
  | Super+1 až Super+5 | přepnutí na plochu 1 až 5 |
  | Super+Shift+1 až Super+Shift+5 | přesun okna na plochu 1 až 5 |
  | Super+šipka vlevo / vpravo | okno do levé / pravé poloviny |
  | Super+šipka nahoru | okno přes celou plochu; z poloviny do horní čtvrtiny |
  | Super+šipka dolů | obnovení okna, minimalizace; z poloviny do dolní čtvrtiny |
  | Super+D | rozložení všech oken plochy vedle sebe |
  | Super+Q | zavření okna |
  | Super+L | zamknutí obrazovky |
  | Super+E | soubory |

  Okna mezi sebou a od okrajů obrazovky drží mezeru 10 px. Tažením okna k okraji obrazovky
  se okno přichytí k polovině nebo do rohu. Zkratky ploch fungují v rozložení us i cz.
- **Vyhledávání aplikací** podle návrhu: uprostřed obrazovky, nejvýš šest výsledků, při
  krátkém dotazu jen aplikace začínající zadanými písmeny, častěji spouštěné výš. Tab přepne
  na soubory a příkazy (spuštění příkazu, v terminálu, soubory v domovské složce). Hlavní
  terminál je jeden: Xfce Terminal. Kontejnery se jmenují Ubuntu a Kali Linux.
- **Horní panel** plave 10 px od horního okraje a od stran obrazovky, mezery kolem pruhu
  jsou průhledné, bez stínu. Indikátory mají zvýraznění při najetí a vlastní popisky.
  Wi-Fi ukazuje název sítě a sílu signálu čtyřmi čárkami.
- **Kalendář** po kliknutí na hodiny: větší a přehlednější, dnešní den v šestiúhelníku,
  kolečko myši nebo šipky přepínají měsíce.
- **Wi-Fi** po kliknutí na síť v panelu: seznam dostupných sítí se silou signálu, kliknutí
  na síť a heslo v okně uprostřed obrazovky, spojení se rovnou aktivuje (bez `nmtui`).
  Špatné heslo nechá okno otevřené a profil s ním se neuloží. Skrytá síť a pokročilé
  nastavení jsou dole v seznamu. Ikona nm-applet v liště už není potřeba.
- **Přihlašovací a zamykací obrazovka** podle návrhu: velké logo, velké hodiny, datum,
  uživatel, heslo, volba sezení, přepnutí uživatele a vypnutí. Zamyká se přes light-locker,
  heslo ověřuje LightDM. Kdyby nová obrazovka nešla spustit, nastoupí původní.
- **Síť:** vedle NetworkManageru běžel i systemd-networkd a dával rozhraním druhou adresu.
  Po restartu síť řídí jen NetworkManager.
- **Šestiúhelníky** v panelu, kalendáři a na zamykací obrazovce mají pravidelný tvar jako logo.

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
