<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/castleparcel-banner-dark.png">
    <img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/castleparcel-banner.png" alt="CastleParcel" width="100%">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/junkerderprovinz/castleparcel/actions/workflows/build.yml"><img src="https://img.shields.io/github/actions/workflow/status/junkerderprovinz/castleparcel/build.yml?branch=main&label=Build&style=for-the-badge&logo=githubactions&logoColor=white" alt="Build" height="36"></a>&nbsp;
  <a href="https://github.com/junkerderprovinz/castleparcel/actions/workflows/lint.yml"><img src="https://img.shields.io/github/actions/workflow/status/junkerderprovinz/castleparcel/lint.yml?branch=main&label=Lint&style=for-the-badge&logo=githubactions&logoColor=white" alt="Lint" height="36"></a>&nbsp;
  <a href="https://hub.docker.com/r/junkerderprovinz/castleparcel"><img src="https://img.shields.io/docker/pulls/junkerderprovinz/castleparcel?style=for-the-badge&logo=docker&logoColor=white&label=Pulls&color=1d99f3" alt="Docker Pulls" height="36"></a>&nbsp;
  <a href="https://hub.docker.com/r/junkerderprovinz/castleparcel"><img src="https://img.shields.io/docker/image-size/junkerderprovinz/castleparcel/latest?style=for-the-badge&logo=docker&logoColor=white&label=Size&color=1d99f3" alt="Image Size" height="36"></a>&nbsp;
  <a href="https://github.com/junkerderprovinz/castleparcel/pkgs/container/castleparcel"><img src="https://img.shields.io/badge/Arch-amd64%20%7C%20arm64-success?style=for-the-badge&logo=linux&logoColor=white" alt="Arch" height="36"></a>&nbsp;
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" height="36"></a>&nbsp;
  <a href="https://ca.unraid.net/apps?q=castleparcel"><img src="https://img.shields.io/badge/Unraid-Template-f15a2c?style=for-the-badge&logo=unraid&logoColor=white" alt="Unraid" height="36"></a>&nbsp;
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-AGPL--3.0-blue?style=for-the-badge&logo=gnu&logoColor=white" alt="License: AGPL-3.0" height="36"></a>
</p>

<p align="center">
CastleParcel is a self-hosted drop zone for your files, with <b>castle walls</b> around it. Think of the
site everyone uses to send big files, only on your own server, with no size paywall, and sealed so tight
that not even that server can look inside. Drop your
files (one or a whole batch), set how long they live (plus an optional password or download
limit), and share a short link or QR code. Zero-knowledge end-to-end encryption, resumable
uploads, inline previews, one tiny container. No accounts, no clouds, no tracking, no nonsense.
</p>

<br>

<p><a id="table-of-contents" href="#table-of-contents"><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/sections/table-of-contents-dark.svg"><img src=".github/assets/sections/table-of-contents.svg" alt="Table of Contents" width="100%"></picture></a></p>

1. [Download](#1-download)
2. [Support](#2-support)
3. [What it looks like](#3-what-it-looks-like)
4. [What it does](#4-what-it-does)
5. [Threat model](#5-threat-model)
6. [Getting started](#6-getting-started)
7. [How AI is used here](#7-how-ai-is-used-here)

<br>

<p><a id="1-download" href="#1-download"><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/sections/1-download-dark.svg"><img src=".github/assets/sections/1-download.svg" alt="1. Download" width="100%"></picture></a></p>

<!-- download-buttons: written by scripts/gen_download_buttons.py -->
<p align="center">
  <picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/sections/group-server-dark.svg"><img src=".github/assets/sections/group-server.svg" alt="Server" width="100%"></picture>
</p>
<p align="center">
  &nbsp;<a href="https://ca.unraid.net/apps?q=castleparcel"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(0,-20.153,841.9,285.606))" alt="Install from Unraid&#x27;s Community Applications" width="160" height="54.278"></a>&nbsp;
  &nbsp;<a href="https://mos-official.net/apps.html?search=castleparcel"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(866,-20.153,841.9,285.606))" alt="Install from the MOS Hub" width="160" height="54.278"></a>&nbsp;
  &nbsp;<a href="https://github.com/junkerderprovinz/zimaos-apps#install"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(1732,-20.153,841.9,285.606))" alt="Install from the ZimaOS and CasaOS store" width="160" height="54.278"></a>&nbsp;
  &nbsp;<a href="https://github.com/junkerderprovinz/umbrel-apps#install"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(2598,-20.153,841.9,285.606))" alt="Install from the umbrelOS community app store" width="160" height="54.278"></a>&nbsp;
  <br>
  &nbsp;<a href="https://github.com/junkerderprovinz/truenas-apps#castleparcel"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(3464,-20.153,841.9,285.606))" alt="Install on TrueNAS with a ready-made YAML file" width="160" height="54.278"></a>&nbsp;
  &nbsp;<a href="#proxmox-ve"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(4330,-20.153,841.9,285.606))" alt="Run it on Proxmox VE" width="160" height="54.278"></a>&nbsp;
  &nbsp;<a href="#synology"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(5196,-20.153,841.9,285.606))" alt="Run it on a Synology NAS" width="160" height="54.278"></a>&nbsp;
  &nbsp;<a href="#qnap"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(6062,-20.153,841.9,285.606))" alt="Run it on a QNAP NAS" width="160" height="54.278"></a>&nbsp;
  <br>
  &nbsp;<a href="#ugreen"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(6928,-20.153,841.9,285.606))" alt="Run it on a UGREEN NAS" width="160" height="54.278"></a>&nbsp;
  &nbsp;<a href="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/compose.yaml"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(7794,-20.153,841.9,285.606))" alt="Download the docker-compose file" width="160" height="54.278"></a>&nbsp;
  &nbsp;<a href="https://github.com/junkerderprovinz/castleparcel/releases/latest"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(8660,-20.153,841.9,285.606))" alt="Download the source archive" width="160" height="54.278"></a>&nbsp;
</p>
<!-- /download-buttons -->

<br>

<p><a id="2-support" href="#2-support"><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/sections/2-support-dark.svg"><img src=".github/assets/sections/2-support.svg" alt="2. Support" width="100%"></picture></a></p>

<p align="center">
A one-knight job: I build it, keep it running, work through the issues and add what people ask for, until nothing is missing. It is free, with no accounts, no telemetry, no ads and no paid tier. No asterisk anywhere. Nothing readable ever leaves your own walls. Forged on evenings and weekends, with heart and stubbornness.
</p>

<p align="center">
If it has earned a place on your server or computer, toss a coin to your knight: it helps cover the costs and keeps the project alive. It also makes this knight's heart beat a little faster. Three ways below, whichever suits you.
</p>

<!-- give-buttons: written by scripts/gen_download_buttons.py -->
<p align="center">
  &nbsp;<a href="https://buymeacoffee.com/junkerderprovinz"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(9526,-20.153,841.9,285.606))" alt="Buy me a coffee" width="160" height="54.278"></a>&nbsp;
  &nbsp;<a href="https://www.paypal.com/donate/?hosted_button_id=76FVV52TKXTUS"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(10392,-20.153,841.9,285.606))" alt="PayPal" width="160" height="54.278"></a>&nbsp;
  &nbsp;<a href="https://junkerderprovinz.github.io/junkerderprovinz/"><img src="https://raw.githubusercontent.com/junkerderprovinz/castleparcel/main/.github/assets/download-buttons/buttons.svg?v=2659e611c06d#svgView(viewBox(11258,-20.153,841.9,285.606))" alt="Donate with crypto" width="160" height="54.278"></a>&nbsp;
</p>
<!-- /give-buttons -->

<br>

<p><a id="3-what-it-looks-like" href="#3-what-it-looks-like"><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/sections/3-what-it-looks-like-dark.svg"><img src=".github/assets/sections/3-what-it-looks-like.svg" alt="3. What it looks like" width="100%"></picture></a></p>

The files and links in these pictures are made up.

<p align="center">
  <img src=".github/assets/screenshots/castleparcel-1.png" alt="CastleParcel in a browser window: a photo is selected and the options for expiry, download limit, password and photo metadata are shown" width="100%">
  <br><em>Pick the files, then decide how long they live and how often they can be downloaded</em>
</p>

<p align="center">
  <img src=".github/assets/screenshots/castleparcel-2.png" alt="CastleParcel showing the finished share link with a QR code and the management link" width="100%">
  <br><em>The link is ready: copy it or save the QR code</em>
</p>

<p align="center">
  <img src=".github/assets/screenshots/castleparcel-3.png" alt="The share page a recipient opens, with the files, their sizes, the expiry and a download button" width="100%">
  <br><em>What the recipient sees, decrypted in their browser</em>
</p>

<br>

<p><a id="4-what-it-does" href="#4-what-it-does"><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/sections/4-what-it-does-dark.svg"><img src=".github/assets/sections/4-what-it-does.svg" alt="4. What it does" width="100%"></picture></a></p>

- **Zero-knowledge.** Files are encrypted in your browser before they upload (libsodium XChaCha20-Poly1305), names included. The key lives in the link's `#fragment` or comes from the share's password, so the server and anyone with its disk only ever see ciphertext.
- **Shares that clean up after themselves.** Expiry from one hour to 30 days or never, an optional download limit and an optional password. A download counts as soon as it starts, and one that breaks off picks up where it stopped, so a shaky connection doesn't waste it. Recipients of a limited link see the file names and sizes before they spend one. A private management link lets you delete a share early. Several files travel as one bundle under one link. The recipient saves them all as one ZIP, built in the browser while it decrypts, and on a share without a download limit can also pick single files.
- **Notes and file requests.** Share a text note with the same encryption and limits as a file. Or ask someone for files: they get a link to send you files without an account, and only your private collect link can open what arrives.
- **Big files are fine.** Uploads go in resumable 64 MiB chunks, so a dropped connection picks up where it stopped and a size-capped CDN such as Cloudflare does not get in the way.
- **Photos travel light.** JPEGs lose their EXIF and GPS data in the browser before encryption, unless you switch it off.
- **Pleasant to use.** Paste a screenshot with Ctrl+V, install it as an app and share to it from Android, preview images, video, audio, text and PDF before downloading, and read it all in 42 languages.
- **Safe to put online.** Per-IP rate limits, an optional daily upload cap per visitor, a storage quota, an expiry cap and an optional upload password. A strict Content-Security-Policy keeps foreign scripts off the page, and share pages tell search engines to stay away.
- **Owner sign-in, if you want it.** Without a password CastleParcel stays open as before. Set one under Settings and only you send files, plus guests who know the upload password. Two-factor authentication, passkeys, API tokens and sign-in services such as Pocket ID or authentik come with it. Recipients never sign in.
- **Tells you what happened.** Notifications through ntfy, Gotify, Discord, Slack, Matrix, email, Apprise or a webhook when files arrive at a file request, when the store is almost full or when the password was reset.
- **One small container.** A static Go binary on a distroless base with a single SQLite file, for amd64 and arm64, with SBOM and provenance attestations and a CVE scan on every build.

It has no user accounts beyond the owner, no LDAP, no email and no S3 storage. If you need those, [Pingvin Share](https://github.com/stonith404/pingvin-share) is the bigger tool that inspired this one.

<br>

<p><a id="5-threat-model" href="#5-threat-model"><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/sections/5-threat-model-dark.svg"><img src=".github/assets/sections/5-threat-model.svg" alt="5. Threat model" width="100%"></picture></a></p>

Everything is encrypted in your browser, so the server only ever holds ciphertext. That protects a lot, but not everything, and one part still comes down to trust. Here is where the lines are.

The server never sees your files, their names or types, or the password in readable form. The key travels in the `#` part of the link, and browsers do not send that part to the server. Shares in the v2 format pad their sizes with Padmé, so the server learns a size bucket instead of the exact size or the number of files. Every new share is written in v2. Older shares stay readable, and for those the server sees the exact size, while one protected by a password alone can be guessed offline by anyone who has its slug. [docs/encryption-format.md](docs/encryption-format.md) covers both in detail.

Whoever runs the server can still see visitors' IP addresses, when a share was uploaded and downloaded and how often, its expiry and download limit, and the size of the encrypted file. With HTTPS, someone watching the network sees that you talk to the server, when, and roughly how much data moves. Without HTTPS they can also change the page on its way to you, so put HTTPS in front of any instance you use for real. If the operator sets `APP_LOGO` to an outside URL, that host sees your IP address as well.

The weak spot is the JavaScript. Your browser runs whatever code the server sends, and a server that was broken into, or a tampered image, could send code that reads your key or your files before they are encrypted. It could even do that for a few visitors only. No web app can rule this out by itself, so CastleParcel makes the code checkable instead:

- The client builds to the same bytes on any machine. CI builds it twice on Linux, Windows and macOS and compares the results.
- Every release carries `castleparcel-client.json`, with the SHA-256 of every file of the client and one root hash over all of them.
- Each instance serves the manifest of the client it embeds at `/.well-known/castleparcel-client.json` and prints the root hash in its log at startup.
- `index.html` pins its script and stylesheet with Subresource Integrity, so the browser refuses a file that does not match. The page itself is the release's template with the instance's name filled in.
- The crypto worker, which does the encryption, decryption and key derivation of every share format, is part of that pinned script. The page starts it from a `blob:` URL made from the script's own code, so there is no separate worker file to check.
- The service worker `sw-download.js` cannot be pinned that way, and it handles downloads after they are decrypted. It is in the manifest like every other file, so the check below covers it.

To check an instance against a release, run this from a checkout of this repository (Node 20 or newer):

```sh
curl -LO https://github.com/junkerderprovinz/castleparcel/releases/download/vX.Y.Z/castleparcel-client.json
node scripts/verify-client.mjs https://share.yourdomain.tld castleparcel-client.json
```

The script fetches every file, compares its hash with the release and makes sure the page differs from the template only by escaped text where the template has placeholders. For an image you have not started, `docker run --rm junkerderprovinz/castleparcel:latest -client-manifest` prints the manifest of the client inside it. [docs/verifying-the-client.md](docs/verifying-the-client.md) explains the format and how to rebuild a release yourself.

A check proves what the instance sent to you at that moment. It says nothing about what the server sends to someone else, and your browser does not compare the code with the release on its own.

The full link is the key, so anyone who has it can open the share until it expires or runs out of downloads. A few habits help:

- Send the link over a channel you trust, and the password, if you set one, over a different one.
- A chat or mail service keeps the whole link, key included, in its copy of your message. Browsers keep it in their history and may sync it.
- Pick a short expiry or a download limit, and delete the share with the management link once it has arrived. Keep the management link to yourself.

Some things are out of reach for any file sharer: malware or a hostile browser extension on the sender's or the recipient's device, a recipient who passes the link on, an operator who deletes shares or shuts the server down, and someone who watches the network long enough to match uploads to downloads.

<br>

<p><a id="6-getting-started" href="#6-getting-started"><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/sections/6-getting-started-dark.svg"><img src=".github/assets/sections/6-getting-started.svg" alt="6. Getting started" width="100%"></picture></a></p>

On Unraid, install **CastleParcel** from [Community Applications](https://ca.unraid.net/apps?q=castleparcel). Anywhere else:

```sh
docker run -d --name castleparcel -p 3000:3000 \
  -e BASE_URL=https://share.yourdomain.tld \
  -e CONFIG_DIR=/config \
  -v /path/to/data:/data \
  -v /path/to/config:/config \
  junkerderprovinz/castleparcel:latest
```

Then open port 3000 and drop a file. `/data` holds the uploads and `/config` the small database, so the two can sit on different disks. To keep both in `/data`, set `CONFIG_DIR=/data` and leave out the `/config` mount.

Put HTTPS in front of it before you share links outside your network. The browser needs a secure context, HTTPS or localhost, for streaming downloads, resuming an upload that broke off, and the clipboard. Over plain HTTP uploads still work, but one that breaks off starts over. Set `BASE_URL` to the public address so the links point there, and in Nginx or Nginx Proxy Manager let big uploads through:

```nginx
client_max_body_size 0;
proxy_read_timeout 3600s;
proxy_send_timeout 3600s;
proxy_request_buffering off;
```

On an instance anyone can reach, set a password under Settings, Sign-in. From then on only you send files, and every share link still works for its recipient. To let others send too, give them the `UPLOAD_PASSWORD`. While no password is set, `UPLOAD_PASSWORD` also guards the Sign-in tab, so nobody without it can set the first one. Passkeys and sign-in services need `BASE_URL`. Forgot the password? The sign-in page takes a code from the container log.

Behind a reverse proxy, set `TRUST_PROXY` to the proxy's address so the rate limits count real visitors instead of the proxy.

### Settings

Everything is set through environment variables. A value the server cannot read stops it at boot with a message that names the variable.

| Variable | Default | What it does |
| --- | --- | --- |
| `BASE_URL` | none | Public address for share links. Without it, links use the address the uploader opened. Passkeys and sign-in services need it. |
| `DATA_DIR` | `/data` | Where the uploaded files live. |
| `CONFIG_DIR` | `/config` | Where the SQLite database and the sign-in key live. |
| `PORT` | `3000` | Port the server listens on. |
| `MAX_FILE_SIZE` | `0` | Largest upload in bytes. `0` means no limit. |
| `DEFAULT_EXPIRY` | `7d` | Expiry picked for a new share: `1h`, `6h`, `1d`, `7d`, `30d` or `never`. |
| `MAX_EXPIRY` | none | Longest expiry an uploader can choose, from the same values. |
| `STORAGE_QUOTA` | `0` | Bytes all shares together may use, uploads in progress included. `0` means no limit. |
| `UPLOAD_PASSWORD` | none | Password guests need to upload. Until you set an owner password, everyone needs it. The signed-in owner never does, and downloads never ask for it. |
| `CASTLEPARCEL_APP_KEY` | none | Key the sign-in is sealed with, 64 hexadecimal characters as `openssl rand -hex 32` prints them. Without it CastleParcel keeps one in `app.key` in `CONFIG_DIR`. |
| `UPLOAD_LIMIT_PER_IP` | `0` | Bytes one visitor may upload in 24 hours. `0` means no limit. IPv6 visitors count per /64. The count stays in memory under a hash that changes with every restart, so the server never keeps an address. |
| `RATE_LIMIT` | `true` | Per-visitor limits on creating uploads, on new downloads and on failed share lookups. |
| `TRUST_PROXY` | `false` | Which reverse proxies may name the visitor in `X-Forwarded-For`. `true` trusts any, or list the proxies as addresses and CIDR ranges, such as `172.16.0.0/12,10.0.0.5`. |
| `APP_NAME` | `CastleParcel` | Name shown on the page and in its title. |
| `APP_LOGO` | none | Logo as an `https://` URL or an image `data:` URL. The page allows images from that URL's host and nothing else from it, and visitors' browsers load the logo from there. |
| `ACCENT_COLOR` | `#d4af37` | Accent colour as a six-digit hex value. |

### Coming from featherdrop

CastleParcel used to be called featherdrop. The share formats, the database and the links stayed the same, so an existing install moves over with all its shares and every link it ever sent keeps working.

- **Unraid:** edit the featherdrop container, set **Repository** to `junkerderprovinz/castleparcel:latest` and apply. Leave the paths as they are.
- **docker run:** start `junkerderprovinz/castleparcel:latest` with the same `/data` and `/config` folders.
- **Compose:** change the image to `junkerderprovinz/castleparcel:latest`. If you take the new [compose.yaml](compose.yaml) instead, stop the old featherdrop container first, since both want port 3000. The new file uses the same `./appdata` folders as the old one.

The featherdrop images stay on Docker Hub and GHCR, but they get no more updates.

### Other Docker hosts

[compose.yaml](compose.yaml) runs CastleParcel on any Linux server with Docker. Put it in a folder of its own and start it with `docker compose up -d`. The folders are created next to the file.

- **Synology:** see [Synology](#synology) below.
- **QNAP:** see [QNAP](#qnap) below.
- **UGREEN:** see [UGREEN](#ugreen) below.
- **Proxmox:** see [Proxmox VE](#proxmox-ve) below.
- **MOS:** install CastleParcel from the [MOS Hub](https://mos-official.net/apps.html?search=castleparcel).
- **ZimaOS and CasaOS:** add the [community store](https://github.com/junkerderprovinz/zimaos-apps#install) and install CastleParcel from it.
- **umbrelOS:** add the [community app store](https://github.com/junkerderprovinz/umbrel-apps#install) and install CastleParcel from it.

### Proxmox VE

CastleParcel runs on Proxmox VE 9.2 in three ways, all tested there. The lightest is a script that installs CastleParcel right in a Debian container. Without the Docker layer it needs less memory and disk, and Proxmox backs it up together with its container. Run it in the shell of the Proxmox host:

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/junkerderprovinz/proxmox-apps/main/ct/castleparcel.sh)"
```

To update later, run `update` in the container's console. Browsers only encrypt on a secure page, so put CastleParcel behind a reverse proxy with HTTPS.

From the image, Proxmox 9.1 and newer turn the container image into a container of its own. Without the Docker layer, CastleParcel needs less memory and disk, and Proxmox backs it up together with its container. Its data stays in folders under `/srv/castleparcel` on the host. Proxmox still calls this a technology preview.

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/junkerderprovinz/proxmox-apps/main/image.sh) install castleparcel
```

To update, run the same line with `update` in place of `install`. It replaces the container and keeps the folders and the address.

Docker works just as well, nothing is wrong with it. Create a container with Docker using the Community Scripts and start the compose file in it, as under [Other Docker hosts](#other-docker-hosts):

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/community-scripts/ProxmoxVE/main/ct/docker.sh)"
```

For a virtual machine instead, take `vm/docker-vm.sh` from the same place.

### Synology

In Container Manager (DSM 7.2 or newer), open **Project** and click **Create**. Pick a folder for CastleParcel and upload [compose.yaml](compose.yaml). Container Manager does not create missing folders, so create the ones the file names under `volumes` in File Station first.

### QNAP

In Container Station, open **Applications** and click **Create**. Paste [compose.yaml](compose.yaml) and point the folders under `volumes` at a share, for example `/share/Container/castleparcel`.

### UGREEN

In the Docker app of UGOS Pro, open **Project** and click **Create**. Pick a folder for CastleParcel and paste [compose.yaml](compose.yaml).

<sub>The Synology logo is a trademark of Synology Inc. and the QNAP logo a trademark of QNAP Systems, Inc., both registered in Taiwan and other regions.</sub>

### Updating

A container does not replace itself, so you update CastleParcel where you installed it. **Check for updates** under **Settings, Info** says when a newer release is out. Shares, links and settings live in `/data` and `/config` and stay as they are.

- **Unraid:** on the **Docker** tab, choose **Check for Updates**, then **apply update** beside CastleParcel.
- **docker run:** run `docker pull junkerderprovinz/castleparcel:latest`, remove the container and start it again with the same command.
- **Compose:** run `docker compose pull`, then `docker compose up -d`, in the folder that holds the file.
- **Other hosts:** update the app on the host's app page, which pulls the newer image.

<br>

<p><a id="7-how-ai-is-used-here" href="#7-how-ai-is-used-here"><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/sections/7-how-ai-is-used-here-dark.svg"><img src=".github/assets/sections/7-how-ai-is-used-here.svg" alt="7. How AI is used here" width="100%"></picture></a></p>

One knight builds this, and AI is one of the tools I work with, the same way I work with an editor or a compiler. It helps me write code and documentation and it checks my work, and that saves me a good many evenings. It does not make the decisions, though. I read and understand everything before it ships, and if something here breaks, that is on me and not on the tool.

You do not have to take my word for it. The code is open and every release note is written by hand. The issue tracker shows how problems actually get handled, including the ones I got wrong the first time. If you find something that is not right, open an issue and I will look at it.

<br>

Questions, bugs, ideas or feature requests? Please [open a GitHub issue](https://github.com/junkerderprovinz/castleparcel/issues).

<sub>The name CastleParcel, its logo and its branding are not covered by the AGPL-3.0 licence of the code: a fork needs its own name and look.</sub><br>
<sub>The licences of the libraries it ships with: every instance serves those of the web client at `/third-party-licenses.md`, and the image keeps all of them under `/licenses`.</sub>
