# ControlPanelVPS — Vollständiges Handbuch

> Version: Phase 1 | Sprache: Deutsch | Betriebssystem: Ubuntu 22.04 / 24.04, Debian 12

---

## Inhaltsverzeichnis

1. [Systemübersicht](#1-systemübersicht)
2. [Installation](#2-installation)
3. [Erste Schritte](#3-erste-schritte)
4. [Server-Verwaltung](#4-server-verwaltung)
5. [Websites & Domains](#5-websites--domains)
6. [SSL/TLS-Zertifikate](#6-ssltls-zertifikate)
7. [Datenbanken](#7-datenbanken)
8. [DNS-Verwaltung](#8-dns-verwaltung)
9. [E-Mail-Server](#9-e-mail-server)
10. [Firewall](#10-firewall)
11. [Backups](#11-backups)
12. [Systemdienste](#12-systemdienste)
13. [Cron Jobs](#13-cron-jobs)
14. [Log-Viewer](#14-log-viewer)
15. [Datei-Manager](#15-datei-manager)
16. [Terminal](#16-terminal)
17. [Benutzerverwaltung & 2FA](#17-benutzerverwaltung--2fa)
18. [Updates](#18-updates)
19. [Einstellungen](#19-einstellungen)
20. [Multi-Server-Betrieb](#20-multi-server-betrieb)
21. [Sicherheit & Härtung](#21-sicherheit--härtung)
22. [Fehlerbehebung](#22-fehlerbehebung)
23. [API-Referenz](#23-api-referenz)
24. [Glossar](#24-glossar)

---

## 1. Systemübersicht

### Was ist ControlPanelVPS?

ControlPanelVPS ist ein selbst-gehostetes, modernes Control Panel zur Verwaltung von Linux-Servern. Es ersetzt kommerzielle Produkte wie Plesk oder cPanel mit einer freien, erweiterbaren Lösung.

### Architektur

```
┌─────────────────────────────────────────────┐
│              Browser (HTTPS)                 │
└────────────────────┬────────────────────────┘
                     │
┌────────────────────▼────────────────────────┐
│            MASTER NODE                       │
│  ┌─────────────────┐   ┌─────────────────┐  │
│  │  Next.js UI     │   │  Go REST API    │  │
│  │  (Port 3000)    │◄──│  (Port 8080)    │  │
│  └─────────────────┘   └────────┬────────┘  │
│                                  │           │
│  ┌──────────────────────────────┐│           │
│  │  PostgreSQL + Redis          ││           │
│  └──────────────────────────────┘│           │
└──────────────────────────────────┼───────────┘
                                   │ HTTP + Bearer Token
              ┌────────────────────┼─────────────────┐
              ▼                    ▼                  ▼
      ┌───────────────┐  ┌───────────────┐  ┌───────────────┐
      │  Agent        │  │  Agent        │  │  Agent        │
      │  Server 1     │  │  Server 2     │  │  Server N     │
      │  (Port 8087)  │  │  (Port 8087)  │  │  (Port 8087)  │
      └───────────────┘  └───────────────┘  └───────────────┘
```

**Master**: Hostet das Web-UI und die REST-API. Speichert alle Konfigurationen in PostgreSQL.

**Agent**: Leichtes Go-Binary (~8 MB), läuft auf jedem verwalteten Server. Führt System-Befehle aus und liefert Metriken.

**Einzel-Server-Betrieb**: Master und Agent laufen auf demselben Server. Der Agent kommuniziert mit `localhost:8087`.

### Komponenten

| Komponente | Technologie | Port |
|---|---|---|
| Web-UI | Next.js 15, TypeScript, Tailwind CSS | 3000 |
| API | Go 1.22, Chi Router | 8080 |
| Agent | Go 1.22, gopsutil | 8087 |
| Datenbank | PostgreSQL 16 | 5432 |
| Cache/Sessions | Redis 7 | 6379 |
| Reverse Proxy | Nginx | 80/443 |

---

## 2. Installation

### Voraussetzungen

**Server-Anforderungen:**
- Ubuntu 22.04 LTS / Ubuntu 24.04 LTS / Debian 12
- Mindestens 1 GB RAM (2 GB empfohlen)
- Mindestens 20 GB Festplatten-Speicher
- Root-Zugriff (SSH)
- Eine Domain, die auf den Server zeigt (für HTTPS)

**Netzwerk:**
- Port 22 (SSH) — für Installation
- Port 80 (HTTP) — für Let's Encrypt
- Port 443 (HTTPS) — für das Panel
- Port 8087 — nur intern, nicht öffentlich zugänglich

### Schnellinstallation

```bash
# Als root einloggen
ssh root@DEINE-IP

# Installationsskript herunterladen und ausführen
curl -fsSL https://raw.githubusercontent.com/Sirbuschi2003/ControlPanelVPS-/master/deploy/install.sh | bash
```

Das Skript fragt interaktiv:
1. **Panel-Domain** — z.B. `panel.meinedomain.de`
2. **Admin E-Mail** — für Let's Encrypt und Login
3. **Admin Passwort** — mindestens 12 Zeichen

**Was das Skript macht:**
1. Prüft das Betriebssystem
2. Installiert Go, Node.js, PostgreSQL, Redis, Nginx, Certbot, Fail2ban, UFW
3. Klont das Repository nach `/opt/controlpanel`
4. Konfiguriert PostgreSQL und Redis
5. Baut Master-API, Agent und Frontend
6. Richtet systemd-Services ein
7. Konfiguriert Nginx als Reverse Proxy
8. Beantragt Let's Encrypt SSL-Zertifikat
9. Konfiguriert UFW-Firewall
10. Registriert den lokalen Server automatisch im Panel

**Nach der Installation:**
```
Panel-URL:   https://panel.meinedomain.de
Admin-Email: deine@email.de
Passwort:    dein-gewähltes-Passwort
```

### Manuelle Installation (Schritt für Schritt)

Falls das automatische Skript nicht funktioniert:

```bash
# 1. Pakete installieren
apt-get update && apt-get install -y git curl wget postgresql redis-server nginx certbot python3-certbot-nginx fail2ban ufw

# 2. Go installieren
curl -fsSL https://go.dev/dl/go1.22.4.linux-amd64.tar.gz | tar -C /usr/local -xz
export PATH=$PATH:/usr/local/go/bin

# 3. Node.js installieren
curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
apt-get install -y nodejs

# 4. Repository klonen
git clone https://github.com/Sirbuschi2003/ControlPanelVPS- /opt/controlpanel
cd /opt/controlpanel

# 5. PostgreSQL konfigurieren
sudo -u postgres createuser cpanel
sudo -u postgres createdb cpanel -O cpanel
sudo -u postgres psql -c "ALTER USER cpanel WITH PASSWORD 'SICHERES_PASSWORT';"

# 6. Umgebungsvariablen setzen
cp /opt/controlpanel/.env.example /opt/controlpanel/.env
nano /opt/controlpanel/.env  # Werte anpassen

# 7. Backend bauen
cd /opt/controlpanel/master && go build -o /opt/controlpanel/bin/master ./cmd/server
cd /opt/controlpanel/agent  && go build -o /opt/controlpanel/bin/agent  ./cmd/agent

# 8. Frontend bauen
cd /opt/controlpanel/frontend && npm ci && npm run build

# 9. Services starten
systemctl start cpanel-master cpanel-agent
```

### Konfigurationsdatei (.env)

```env
# Pflichtfelder
DATABASE_URL=postgres://cpanel:PASSWORT@localhost:5432/cpanel?sslmode=disable
REDIS_URL=redis://:REDIS_PASSWORT@localhost:6379/0
JWT_SECRET=mind_32_zufällige_zeichen_hier
AGENT_TOKEN=geheimes_agent_token

# Optional
LISTEN_ADDR=:8080
ENVIRONMENT=production
PANEL_DOMAIN=panel.meinedomain.de
```

**Wichtig:** Die `.env`-Datei enthält Geheimnisse. Berechtigungen:
```bash
chmod 600 /opt/controlpanel/.env
chown cpanel:cpanel /opt/controlpanel/.env
```

---

## 3. Erste Schritte

### Login

1. Browser öffnen: `https://panel.meinedomain.de`
2. E-Mail und Passwort eingeben
3. Bei aktivierter 2FA: 6-stelligen Code aus der Authenticator-App eingeben

### Dashboard

Das Dashboard zeigt:
- **Server-Status** — Online/Offline-Anzeige aller verwalteten Server
- **Schnellzugriff** — Links zu den wichtigsten Modulen
- **Statistiken** — Gesamtzahl Server, aktive Dienste

### Oberfläche

**Sidebar (links):**
- Klappbar über den Pfeil-Button
- Aktiver Eintrag wird blau hervorgehoben
- Alle Module sind direkt erreichbar

**Topbar (oben rechts):**
- Angemeldeter Benutzer
- Abmelde-Button

---

## 4. Server-Verwaltung

### Server hinzufügen

**Automatisch (bei Installation):** Der lokale Server wird automatisch registriert.

**Manuell über das Panel:**
1. → **Server** → **Server hinzufügen**
2. Felder ausfüllen:
   - **Name**: Beliebiger Anzeigename (z.B. "Web Server 1")
   - **Hostname**: Vollständiger Hostname (z.B. `server1.meinedomain.de`)
   - **IP-Adresse**: Öffentliche IP des Servers
   - **Agent URL**: `http://IP:8087` oder `http://hostname:8087`
   - **Agent Token**: Das Token aus der `.env`-Datei des Servers
3. Speichern

**Agent auf neuem Server installieren:**
```bash
curl -fsSL https://raw.githubusercontent.com/Sirbuschi2003/ControlPanelVPS-/master/deploy/install-agent.sh | bash -s -- \
  --master https://panel.meinedomain.de \
  --token DEIN_AGENT_TOKEN
```

### Server-Metriken

Auf der Server-Karte werden angezeigt:
- **CPU**: Aktuelle Auslastung mit Farbindikator (grün < 70%, gelb < 90%, rot ≥ 90%)
- **RAM**: Verbrauch in Echtzeit
- **Disk**: Festplattennutzung
- **Uptime**: Wie lange der Server läuft
- **Load Average**: Systemlast der letzten 1/5/15 Minuten
- **OS**: Betriebssystem und Kernel-Version

### Server-Status

| Status | Bedeutung |
|---|---|
| Online (grün, pulsierend) | Agent antwortet normal |
| Offline (rot) | Agent nicht erreichbar |
| Unbekannt (gelb) | Noch kein Check durchgeführt |

---

## 5. Websites & Domains

### Website erstellen

1. → **Websites** → **Website hinzufügen**
2. Felder:
   - **Server**: Auf welchem Server die Website laufen soll
   - **Domain**: Primäre Domain (z.B. `meinedomain.de`)
   - **Aliases**: Weitere Domains, kommagetrennt (z.B. `www.meinedomain.de`)
   - **PHP-Version**: 7.4 / 8.1 / 8.2 / 8.3
   - **Document Root**: Pfad zum Web-Verzeichnis (Standard: `/var/www/domain.de/public_html`)

**Was im Hintergrund passiert:**
- Nginx-Konfiguration wird erstellt
- Document-Root-Verzeichnis wird angelegt
- Nginx wird neu geladen
- Website ist sofort erreichbar

### Generierte Nginx-Konfiguration

Für `meinedomain.de` wird erstellt:
```nginx
# /etc/nginx/sites-available/meinedomain.de.conf
server {
    listen 80;
    server_name meinedomain.de www.meinedomain.de;
    root /var/www/meinedomain.de/public_html;
    index index.php index.html;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        fastcgi_pass unix:/run/php/php8.2-fpm.sock;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }
}
```

### SSL aktivieren

**Voraussetzung:** Domain muss auf den Server zeigen (DNS).

1. Erst ein SSL-Zertifikat unter **SSL/TLS** beantragen (siehe Kapitel 6)
2. In der Website-Karte: **SSL aktivieren** → Zertifikat auswählen
3. Optional: **HTTPS erzwingen** aktiviert automatische HTTP→HTTPS-Weiterleitung

### Website aktivieren/deaktivieren

- **Deaktivieren**: Entfernt den Nginx-Symlink → Website ist nicht mehr erreichbar
- **Aktivieren**: Stellt Symlink wieder her → Website wieder online

### Website löschen

- Entfernt Nginx-Konfiguration und Symlink
- **Achtung:** Das Web-Verzeichnis (`/var/www/domain`) wird **nicht** gelöscht
- Dateien müssen manuell über den Datei-Manager entfernt werden

---

## 6. SSL/TLS-Zertifikate

### Kostenloses Zertifikat beantragen (Let's Encrypt)

**Voraussetzung:** Domain zeigt auf diesen Server, Port 80 ist öffentlich erreichbar.

1. → **SSL/TLS** → **Zertifikat beantragen**
2. Felder:
   - **Server**: Auf welchem Server das Zertifikat installiert wird
   - **Domain**: Primäre Domain
   - **Weitere Domains (SANs)**: Subdomains, kommagetrennt (z.B. `www.meinedomain.de`)
   - **E-Mail**: Für Let's Encrypt-Benachrichtigungen
3. **Beantragen** klicken

Der Vorgang dauert 10–30 Sekunden. Let's Encrypt prüft die Domain-Eigentümerschaft automatisch.

**Zertifikat-Speicherort auf dem Server:**
```
/etc/letsencrypt/live/meinedomain.de/
├── fullchain.pem  ← Zertifikat (für Nginx: ssl_certificate)
├── privkey.pem    ← Privater Schlüssel (ssl_certificate_key)
└── chain.pem      ← Zwischenzertifikate
```

### Automatische Verlängerung

Let's Encrypt-Zertifikate sind 90 Tage gültig. Das Panel verlängert sie automatisch:
- Certbot-Systemd-Timer läuft täglich
- Panel erkennt ablaufende Zertifikate (< 30 Tage) und zeigt Warnungen

### Zertifikat-Status

| Status | Bedeutung |
|---|---|
| Aktiv (grün) | Gültig, mehr als 30 Tage |
| Läuft ab (gelb) | Weniger als 30 Tage gültig |
| Kritisch (rot) | Weniger als 7 Tage gültig |
| Fehlgeschlagen | Ausstellung hat nicht geklappt |
| Ausstehend | Wird gerade beantragt |

### Zertifikat manuell erneuern

- In der Zertifikat-Zeile: **Erneuern** klicken
- Oder auf dem Server: `certbot renew --cert-name meinedomain.de`

---

## 7. Datenbanken

### MySQL/MariaDB-Datenbank erstellen

1. → **Datenbanken** → **Datenbank erstellen**
2. Felder:
   - **Server**: Ziel-Server
   - **Name**: Datenbankname (nur Buchstaben, Zahlen, Unterstriche)
   - **Typ**: MySQL / MariaDB / PostgreSQL
   - **Benutzer**: Datenbankbenutzer
   - **Passwort**: Sicheres Passwort (wird verschlüsselt gespeichert)

**Was im Hintergrund passiert (MySQL):**
```sql
CREATE DATABASE `datenbankname`;
CREATE USER `benutzer`@`localhost` IDENTIFIED BY 'passwort';
GRANT ALL PRIVILEGES ON `datenbankname`.* TO `benutzer`@`localhost`;
FLUSH PRIVILEGES;
```

### Passwort abrufen

- **Passwort anzeigen** Button → zeigt das gespeicherte Passwort
- Für Verbindungs-Setup in Anwendungen

### Verbindungsdaten

Typische Verbindungsdaten für PHP/WordPress:
```
Host:     localhost  (oder 127.0.0.1)
Port:     3306
Database: datenbankname
User:     benutzer
Password: dein-passwort
```

### Datenbank löschen

**Achtung:** Löscht die Datenbank und den Datenbankbenutzer unwiderruflich. Vorher Backup erstellen!

---

## 8. DNS-Verwaltung

### Voraussetzung

Der Server muss als Nameserver erreichbar sein. Dafür:
1. Bei deinem Domain-Registrar: NS-Records auf den Server setzen
2. BIND (named) muss auf dem Server installiert sein

### DNS-Zone erstellen

1. → **DNS** → **Zone erstellen**
2. Felder:
   - **Server**: DNS-Server
   - **Domain**: Zone-Name (z.B. `meinedomain.de`)
   - **Nameserver**: Primärer NS (z.B. `ns1.meinedomain.de`)
   - **Admin-E-Mail**: Wird als SOA-Contact gespeichert

**Generierte Zone-Datei:**
```bind
; Zone-Datei für meinedomain.de
$TTL 3600
@   IN  SOA  ns1.meinedomain.de. admin.meinedomain.de. (
        2024010101  ; Serial
        3600        ; Refresh
        900         ; Retry
        604800      ; Expire
        300         ; Minimum TTL
)

@   IN  NS   ns1.meinedomain.de.
```

### DNS-Records hinzufügen

**Häufige Record-Typen:**

| Typ | Verwendung | Beispiel Content |
|---|---|---|
| A | IPv4-Adresse | `185.12.34.56` |
| AAAA | IPv6-Adresse | `2001:db8::1` |
| CNAME | Alias auf anderen Namen | `meinedomain.de.` |
| MX | Mail-Server | `mail.meinedomain.de.` |
| TXT | Texte (SPF, DKIM, etc.) | `v=spf1 mx ~all` |
| SRV | Dienst-Adressen | `10 20 5269 xmpp.meinedomain.de.` |
| CAA | Erlaubte CAs | `0 issue "letsencrypt.org"` |

**TTL**: Time To Live in Sekunden. Standard 3600 (1 Stunde). Für schnelle Updates: 300.

**Priorität (MX)**: Niedrigerer Wert = höhere Priorität. Backup-MX hat höheren Wert.

### Typische DNS-Einrichtung für Webserver

```
meinedomain.de.      A      185.12.34.56
www                  CNAME  meinedomain.de.
mail                 A      185.12.34.56
@                    MX  10 mail.meinedomain.de.
@                    TXT    "v=spf1 mx ~all"
```

---

## 9. E-Mail-Server

### Voraussetzung

Das Panel richtet Postfix + Dovecot mit virtuellen Mailboxen ein. Voraussetzungen:
- Port 25 (SMTP) muss offen sein (bei manchen VPS-Anbietern gesperrt)
- MX-Record für die Domain muss gesetzt sein
- Reverse-DNS (PTR) sollte auf den Hostnamen zeigen

### E-Mail-Domain hinzufügen

1. → **E-Mail** → Tab **Domains** → **Domain hinzufügen**
2. Domain eingeben (z.B. `meinedomain.de`)

### E-Mail-Konto erstellen

1. → **E-Mail** → Tab **Konten** → **Konto erstellen**
2. Felder:
   - **Domain**: Aus welcher Domain
   - **Benutzername**: Teil vor dem @
   - **Passwort**: Sicheres Passwort
   - **Quota**: Maximale Postfachgröße in MB

**Beispiel:** Benutzername `info`, Domain `meinedomain.de` → E-Mail: `info@meinedomain.de`

### E-Mail-Konto einrichten (in Mailprogramm)

```
IMAP-Server:   mail.meinedomain.de
IMAP-Port:     993 (SSL) oder 143 (STARTTLS)
SMTP-Server:   mail.meinedomain.de
SMTP-Port:     587 (STARTTLS) oder 465 (SSL)
Benutzername:  info@meinedomain.de
Passwort:      dein-passwort
```

### Weiterleitungen (Aliases)

1. → **E-Mail** → Tab **Weiterleitungen** → **Weiterleitung erstellen**
2. **Quelle**: `kontakt@meinedomain.de`
3. **Ziel**: `info@meinedomain.de` oder externe Adresse

### SPF, DKIM, DMARC

**SPF** (verhindert E-Mail-Spoofing):
```
TXT-Record: @ → "v=spf1 mx a ip4:DEINE_IP ~all"
```

**DKIM** (Signatur):
Nach der Domain-Erstellung zeigt das Panel den DKIM-Key. Diesen als TXT-Record eintragen:
```
TXT-Record: mail._domainkey → "v=DKIM1; k=rsa; p=DEIN_PUBLIC_KEY"
```

**DMARC** (Policy):
```
TXT-Record: _dmarc → "v=DMARC1; p=quarantine; rua=mailto:dmarc@meinedomain.de"
```

---

## 10. Firewall

Das Panel verwendet **UFW** (Uncomplicated Firewall) als Frontend für iptables.

### Standard-Regeln nach Installation

```
Status: Aktiv
Standard eingehend:  DENY
Standard ausgehend:  ALLOW

Erlaubt:
22/tcp    → SSH
80/tcp    → HTTP
443/tcp   → HTTPS
```

### Firewall-Regel erstellen

1. → **Firewall** → **Regel hinzufügen**
2. Felder:
   - **Aktion**: Allow (erlauben) / Deny (blockieren)
   - **Richtung**: In (eingehend) / Out (ausgehend)
   - **Protokoll**: TCP / UDP / ICMP / Any
   - **Quelle**: IP-Adresse oder `any` für alle
   - **Ziel-Port**: Einzelner Port (z.B. `3306`) oder Bereich (z.B. `8000:9000`)
   - **Kommentar**: Beschreibung der Regel
   - **Reihenfolge**: Niedrigere Zahl = höhere Priorität

**Beispiele:**

| Zweck | Aktion | Protokoll | Quelle | Port |
|---|---|---|---|---|
| Web erlauben | Allow | TCP | any | 80 |
| HTTPS erlauben | Allow | TCP | any | 443 |
| MySQL nur lokal | Deny | TCP | any | 3306 |
| SSH nur von IP | Allow | TCP | 1.2.3.4 | 22 |
| Mail | Allow | TCP | any | 587 |

### Firewall neu laden

Nach Regel-Änderungen wird die Firewall automatisch neu geladen. Manuell:
- **Firewall neu laden** Button im Panel
- Oder: `ufw reload` auf dem Server

### Wichtige Warnung

**Niemals Port 22 (SSH) blockieren ohne eine andere Zugriffsmethode zu haben!** Falls du aus Versehen ausgesperrt wirst, hilft nur der VPS-Anbieter (KVM-Konsole).

---

## 11. Backups

### Backup-Konfiguration erstellen

1. → **Backups** → Tab **Konfigurationen** → **Konfiguration erstellen**
2. Felder:
   - **Server**: Welcher Server gesichert wird
   - **Name**: Bezeichnung der Backup-Konfiguration
   - **Speicher-Typ**: Lokal / S3 / SFTP
   - **Zeitplan**: Cron-Ausdruck (Standard: `0 2 * * *` = täglich 2 Uhr)
   - **Aufbewahrung**: Wie viele Tage Backups behalten
   - **Pfade**: Was gesichert wird (Standard: `/etc`, `/var/www`, `/var/lib/mysql`)
   - **Verschlüsselung**: AES-256-CBC vor dem Speichern

### Speicher-Typen

**Lokal** (`/var/backups/cpanel/`):
- Kein extra Setup nötig
- Backups bleiben auf demselben Server (kein Schutz bei Server-Totalausfall)

**S3 (Amazon S3, Hetzner Object Storage, Backblaze B2, etc.):**
```
S3 Bucket:     mein-backup-bucket
S3 Region:     eu-central-1
Access Key:    AKIA...
Secret Key:    ...
Endpunkt:      https://s3.hetzner.com  (für Hetzner)
```

**SFTP:**
```
Host:     backup-server.example.com
Port:     22
Benutzer: backup
Passwort: ...
Pfad:     /backups/meinserver
```

### Backup manuell starten

- In der Konfigurations-Karte: **Jetzt sichern** klicken
- Status wird in Echtzeit aktualisiert

### Backup-Verlauf

→ **Backups** → Tab **Verlauf**:
- Alle bisherigen Backup-Jobs
- Status (Erfolgreich / Fehlgeschlagen / Läuft)
- Dateigröße
- Start- und Endzeit
- Fehlermeldung bei Fehlschlägen

### Backup-Zeitplan verstehen

```
Cron-Format: Minute Stunde Tag-des-Monats Monat Wochentag

0 2 * * *     = Täglich um 2:00 Uhr
0 2 * * 0     = Wöchentlich, Sonntags 2:00 Uhr
0 2 1 * *     = Monatlich, 1. des Monats 2:00 Uhr
0 */6 * * *   = Alle 6 Stunden
```

### Backup-Verschlüsselung

Wenn aktiviert: Backup wird mit AES-256-CBC verschlüsselt. Der Schlüssel steht in `/opt/controlpanel/.env` als `BACKUP_KEY`. **Diesen Key separat sichern** — ohne ihn können Backups nicht entschlüsselt werden.

Entschlüsseln auf dem Server:
```bash
openssl enc -d -aes-256-cbc -in backup.tar.gz.enc -out backup.tar.gz -pass env:BACKUP_KEY
```

---

## 12. Systemdienste

### Dienste verwalten

→ **Dienste** → Server auswählen

Angezeigte Dienste:
- `nginx` — Web-Server
- `mysql` / `mariadb` — Datenbank
- `postgresql` — Datenbank
- `redis-server` — Cache
- `postfix` — Mail-Transfer
- `dovecot` — Mail-Zugriff (IMAP)
- `fail2ban` — Brute-Force-Schutz
- `php8.x-fpm` — PHP-Prozesse
- `cron` — Cron-Daemon
- `ssh` — SSH-Server

### Aktionen

| Aktion | Wirkung |
|---|---|
| Start | Startet gestoppten Dienst |
| Stop | Stoppt laufenden Dienst |
| Restart | Startet neu (kurze Unterbrechung) |
| Reload | Lädt Konfiguration neu (ohne Unterbrechung, wenn unterstützt) |
| Enable | Autostart beim Server-Boot aktivieren |
| Disable | Autostart deaktivieren |

### Status-Anzeige

- **Grün / active**: Dienst läuft normal
- **Rot / inactive**: Dienst ist gestoppt
- **Grau / unknown**: Dienst nicht installiert oder nicht gefunden

---

## 13. Cron Jobs

Cron Jobs führen Befehle zu bestimmten Zeiten automatisch aus.

### Cron Job erstellen

1. → **Cron Jobs** → **Job erstellen**
2. Felder:
   - **Server**: Auf welchem Server
   - **Name**: Beschreibender Name
   - **Befehl**: Auszuführender Shell-Befehl
   - **Zeitplan**: Cron-Ausdruck
   - **Benutzer**: Unter welchem Systembenutzer ausführen

**Häufige Zeitplan-Presets:**
```
Jede Minute:     * * * * *
Stündlich:       0 * * * *
Täglich 3 Uhr:   0 3 * * *
Wöchentlich:     0 3 * * 1
Monatlich:       0 3 1 * *
```

### Typische Anwendungsbeispiele

```bash
# WordPress-Cron
*/5 * * * *  www-data  php /var/www/wordpress/wp-cron.php

# Let's Encrypt Verlängerung
0 3 * * *  root  certbot renew --quiet

# Datenbankbackup
0 1 * * *  root  mysqldump -u root --all-databases | gzip > /tmp/db_$(date +%F).sql.gz

# Logs aufräumen
0 4 * * 0  root  find /var/log -name "*.gz" -mtime +30 -delete
```

### Cron-Dateien auf dem Server

Das Panel erstellt Dateien in `/etc/cron.d/`:
```
/etc/cron.d/cpanel-JOB-ID
```

Inhalt:
```
# Jobname
*/5 * * * * www-data php /var/www/wordpress/wp-cron.php
```

---

## 14. Log-Viewer

### Verfügbare Logs

| Log-Name | Datei | Inhalt |
|---|---|---|
| `nginx-access` | `/var/log/nginx/access.log` | HTTP-Anfragen |
| `nginx-error` | `/var/log/nginx/error.log` | Nginx-Fehler |
| `syslog` | `/var/log/syslog` | System-Events |
| `auth` | `/var/log/auth.log` | Login-Versuche, SSH |
| `mail` | `/var/log/mail.log` | E-Mail-Aktivität |
| `mysql` | `/var/log/mysql/error.log` | Datenbank-Fehler |
| `fail2ban` | `/var/log/fail2ban.log` | Gesperrte IPs |
| `dpkg` | `/var/log/dpkg.log` | Paket-Installationen |

### Log-Anzeige

1. → **Logs** → Server auswählen → Log auswählen
2. **Zeilen**: 50 / 100 / 200 / 500
3. **Suche**: Filtert Zeilen die den Suchbegriff enthalten
4. **Auto-Refresh**: Aktualisiert alle 10 Sekunden automatisch

### Log-Farben

- **Rot**: ERROR, CRITICAL, FATAL
- **Gelb**: WARN, WARNING, NOTICE  
- **Grün**: INFO, DEBUG
- **Weiß**: Alles andere

### Zugriff per SSH auf Log-Dateien

```bash
# Live-Tail
tail -f /var/log/nginx/access.log

# Suchen
grep "404" /var/log/nginx/access.log | tail -100

# Fehler in letzter Stunde
grep "$(date +%Y/%m/%d\ %H)" /var/log/nginx/error.log
```

---

## 15. Datei-Manager

### Erlaubte Pfade

Aus Sicherheitsgründen ist der Zugriff auf bestimmte Verzeichnisse beschränkt:

| Pfad | Zweck |
|---|---|
| `/var/www` | Web-Verzeichnisse |
| `/etc/nginx` | Nginx-Konfigurationen |
| `/etc/postfix` | Mail-Konfigurationen |
| `/var/log` | Log-Dateien (nur lesen) |
| `/home` | Benutzer-Homeverzeichnisse |
| `/tmp` | Temporäre Dateien |

### Navigation

- **Pfad** oben zeigt aktuellen Pfad mit Breadcrumb-Navigation
- **Klick auf Ordner**: Öffnet Ordner
- **Klick auf Datei**: Öffnet Datei-Editor

### Dateien bearbeiten

1. Auf Dateinamen klicken
2. Inhalt wird im Modal-Editor angezeigt
3. Bearbeiten
4. **Speichern** klicken

**Maximale Dateigröße**: 1 MB (größere Dateien müssen per SSH bearbeitet werden)

### Neuen Ordner erstellen

1. **Neuer Ordner** klicken
2. Pfad eingeben
3. Bestätigen

### Dateien hochladen

1. **Hochladen** klicken
2. Datei auswählen
3. Datei wird als Text-Content gespeichert

**Für binäre Dateien (ZIP, Images) SSH nutzen:**
```bash
scp datei.zip root@SERVER:/var/www/meinedomain.de/
```

---

## 16. Terminal

### Web-SSH

Das Terminal-Modul zeigt Verbindungsinformationen für SSH:

```bash
ssh root@DEINE-IP
# oder
ssh root@panel.meinedomain.de
```

Für Dateiübertragungen:
```bash
scp lokale-datei.txt root@SERVER:/var/www/
scp root@SERVER:/var/www/datei.txt ./lokal/
```

### SSH-Key-Authentifizierung einrichten (empfohlen)

```bash
# Schlüsselpaar erstellen (einmalig, auf deinem lokalen PC)
ssh-keygen -t ed25519 -C "mein-server"

# Öffentlichen Schlüssel auf den Server kopieren
ssh-copy-id root@DEINE-IP

# Ab jetzt ohne Passwort
ssh root@DEINE-IP
```

---

## 17. Benutzerverwaltung & 2FA

### Benutzer verwalten

**Benutzer-Rollen:**
| Rolle | Rechte |
|---|---|
| `admin` | Vollzugriff auf alle Funktionen |
| `viewer` | Nur Lesen (geplant) |

**Neuen Benutzer erstellen:**
1. → **Benutzer** → **Benutzer hinzufügen**
2. E-Mail, Name, Passwort, Rolle ausfüllen
3. Speichern

**Passwort ändern:**
- Admin kann Passwort aller Benutzer ändern
- Eigenes Passwort über → **Benutzer** → Passwort-Icon

### Zwei-Faktor-Authentifizierung (2FA) einrichten

**Empfehlung: 2FA für alle Admin-Konten aktivieren!**

1. → **Benutzer** → **2FA einrichten** (Schlüssel-Icon)
2. QR-Code mit Authenticator-App scannen:
   - **Google Authenticator** (Android/iOS)
   - **Authy** (Android/iOS/Desktop)
   - **Bitwarden** (integrierter TOTP)
   - **1Password** (integrierter TOTP)
3. 6-stelligen Code aus der App eingeben
4. **Aktivieren** klicken

**Beim nächsten Login:**
1. E-Mail + Passwort eingeben
2. 2FA-Code-Feld erscheint
3. Aktuellen 6-stelligen Code eingeben

**Backup-Codes**: Nach 2FA-Aktivierung werden Backup-Codes angezeigt. Sicher aufbewahren! Bei verlorenem Handy kannst du dich damit einloggen.

**2FA deaktivieren:**
- Admin kann 2FA für jeden Benutzer deaktivieren
- → **Benutzer** → 2FA-Icon → Deaktivieren

---

## 18. Updates

### Panel automatisch aktualisieren

1. → **Updates**
2. Aktuellen Commit und Branch werden angezeigt
3. **Auf Updates prüfen** klicken
4. Wenn Update verfügbar: **Update installieren** klicken

**Was beim Update passiert:**
1. `git pull` — neuester Code von GitHub
2. Rebuild Master-API (Go)
3. Rebuild Agent (Go)
4. Rebuild Frontend (Next.js)
5. Neustart der Services (kurze Unterbrechung ~30 Sekunden)

### Update per SSH (manuell)

```bash
ssh root@DEIN-SERVER
bash /opt/controlpanel/deploy/update.sh
```

### Rollback (vorherige Version)

Falls ein Update Probleme macht:

```bash
ssh root@DEIN-SERVER
cd /opt/controlpanel

# Letzten funktionierenden Commit anzeigen
git log --oneline -10

# Auf bestimmten Commit zurücksetzen
git reset --hard COMMIT-HASH

# Neu bauen
bash deploy/update.sh
```

### Auto-Update einrichten (optional)

Cron Job im Panel erstellen:
```
Schedule:  0 3 * * 0    (Sonntags 3 Uhr)
Benutzer:  root
Befehl:    bash /opt/controlpanel/deploy/update.sh
```

---

## 19. Einstellungen

### Panel-Einstellungen

| Einstellung | Beschreibung |
|---|---|
| `panel_name` | Anzeigename des Panels |
| `panel_timezone` | Zeitzone für Anzeigen (z.B. `Europe/Berlin`) |

### E-Mail-Benachrichtigungen

Für Benachrichtigungen bei wichtigen Ereignissen (Backup-Fehler, abgelaufene Zertifikate):

| Einstellung | Beschreibung |
|---|---|
| `smtp_host` | SMTP-Server (z.B. `smtp.gmail.com`) |
| `smtp_port` | SMTP-Port (587 für STARTTLS, 465 für SSL) |
| `smtp_user` | SMTP-Benutzername |
| `smtp_pass` | SMTP-Passwort |
| `smtp_from` | Absender-Adresse |
| `notify_email` | Empfänger-Adresse für Benachrichtigungen |

---

## 20. Multi-Server-Betrieb

### Konzept

Beliebig viele Server können vom selben Panel aus verwaltet werden. Jeder Server braucht nur den leichten Agent.

```
Panel (Master)
├── Server 1 (Web + Agent)
├── Server 2 (Datenbank + Agent)
├── Server 3 (Mail + Agent)
└── Server 4 (Backup + Agent)
```

### Server-Rollen

| Rolle | Verwendung |
|---|---|
| `general` | Allgemeiner Server |
| `web` | Primär Web-Hosting |
| `database` | Primär Datenbanken |
| `mail` | Primär Mail-Server |
| `backup` | Primär Backups |
| `dns` | Primär DNS-Server |

### Sicherheit im Multi-Server-Betrieb

**Agent-Port absichern:**
```bash
# Auf dem Agent-Server: Port 8087 nur für Master-IP erlauben
ufw allow from MASTER_IP to any port 8087 proto tcp comment "ControlPanel Master"
ufw deny 8087
```

**Verschiedene Agent-Tokens pro Server:**
Jeder Server sollte ein eigenes, zufälliges Agent-Token haben:
```bash
openssl rand -hex 24
```

### Fail-Over (automatischer Ausweich-Server)

Geplant für zukünftige Version: Health-Check-basiertes DNS-Failover zwischen Servern.

---

## 21. Sicherheit & Härtung

### Empfohlene Sicherheitsmaßnahmen

#### 1. 2FA aktivieren
Für alle Admin-Konten 2FA einrichten (Kapitel 17).

#### 2. SSH absichern

```bash
# /etc/ssh/sshd_config
PermitRootLogin prohibit-password   # Nur SSH-Key, kein Passwort
PasswordAuthentication no            # Passwort-Login deaktivieren
Port 2222                            # Anderen Port nutzen (optional)
MaxAuthTries 3
```

Nach Änderung: `systemctl restart ssh`

#### 3. Fail2ban-Konfiguration prüfen

```bash
fail2ban-client status               # Übersicht
fail2ban-client status sshd          # SSH-Jail Status
fail2ban-client status nginx-http-auth
```

#### 4. Regelmäßige Backups

- Mindestens täglich
- Auf externen Storage (S3 oder SFTP)
- Verschlüsselung aktivieren

#### 5. Paket-Updates

Über das Panel → **Updates** oder:
```bash
apt-get update && apt-get upgrade -y
```

#### 6. Audit-Log beachten

Alle Panel-Aktionen werden geloggt. Bei verdächtigen Aktivitäten:
- PostgreSQL: `SELECT * FROM audit_log ORDER BY created_at DESC LIMIT 50;`

#### 7. Sichere Passwörter

- Admin-Passwort: Mindestens 16 Zeichen, Buchstaben + Zahlen + Sonderzeichen
- Datenbank-Passwörter: Mindestens 20 Zeichen
- Agent-Tokens: Mindestens 48 Zeichen (hex)

#### 8. Firewall-Regeln regelmäßig prüfen

→ **Firewall** → Alle Regeln überprüfen. Unnötige Ports schließen.

### Sicherheits-Checkliste

```
[ ] 2FA für Admin-Konten aktiviert
[ ] SSH-Key-Authentifizierung (kein Passwort-Login)
[ ] Fail2ban aktiv und konfiguriert
[ ] Tägliche Backups auf externem Speicher
[ ] Firewall aktiviert, nur nötige Ports offen
[ ] Regelmäßige Paket-Updates
[ ] Starke Passwörter (≥ 16 Zeichen)
[ ] Agent-Port 8087 nur für Master-IP zugänglich
[ ] .env-Datei mit Berechtigungen 600
```

---

## 22. Fehlerbehebung

### Panel nicht erreichbar

```bash
# Services prüfen
systemctl status cpanel-master
systemctl status cpanel-agent
systemctl status nginx

# Logs prüfen
journalctl -u cpanel-master -n 50
journalctl -u nginx -n 50

# Port prüfen
ss -tlnp | grep -E '8080|3000|80|443'
```

### Agent nicht verbunden (Server zeigt "Offline")

```bash
# Auf dem Agent-Server
systemctl status cpanel-agent
journalctl -u cpanel-agent -n 50

# Erreichbarkeit testen (vom Master aus)
curl -H "Authorization: Bearer TOKEN" http://AGENT-IP:8087/health

# Firewall prüfen
ufw status | grep 8087
```

### Datenbank-Verbindungsfehler

```bash
# PostgreSQL-Status
systemctl status postgresql

# Verbindung testen
psql -U cpanel -d cpanel -h localhost

# Logs
tail -50 /var/log/postgresql/postgresql-*.log
```

### SSL-Zertifikat-Ausstellung fehlgeschlagen

```bash
# Certbot-Test
certbot certonly --nginx -d meinedomain.de --dry-run

# Häufige Ursachen:
# 1. Domain zeigt nicht auf Server → DNS prüfen
# 2. Port 80 nicht offen → ufw allow 80
# 3. Nginx läuft nicht → systemctl start nginx
# 4. Rate Limit → max 5 Zertifikate pro Domain pro Woche

# Certbot-Logs
journalctl -u certbot
```

### Nginx-Konfigurationsfehler

```bash
nginx -t                      # Konfiguration prüfen
systemctl reload nginx         # Neu laden
tail -50 /var/log/nginx/error.log
```

### Backup schlägt fehl

```bash
# Backup-Verzeichnis prüfen
ls -la /var/backups/cpanel/
df -h                          # Speicherplatz prüfen

# S3-Verbindung testen
aws s3 ls s3://mein-bucket --region eu-central-1

# Manuell testen
tar -czf /tmp/test.tar.gz /etc 2>&1
```

### Passwort vergessen

```bash
# Auf dem Server als root
cd /opt/controlpanel
# Neues Passwort-Hash generieren
htpasswd -bnBC 12 "" NEUES_PASSWORT | tr -d ':\n'

# In der Datenbank aktualisieren
psql -U cpanel -d cpanel -c "UPDATE users SET password = 'HASH' WHERE email = 'admin@panel.local';"
```

### Services nach Server-Neustart nicht gestartet

```bash
# Services aktivieren
systemctl enable cpanel-master cpanel-agent nginx postgresql redis-server

# Neustart-Test
systemctl reboot
# Nach Neustart:
systemctl status cpanel-master cpanel-agent
```

---

## 23. API-Referenz

Alle Endpunkte erfordern `Authorization: Bearer TOKEN` (außer `/api/auth/login`).

### Authentifizierung

```
POST /api/auth/login
Body: {"email": "...", "password": "...", "totp_code": "..."}
→ {"token": "JWT_TOKEN", "user": {...}}

GET /api/auth/me
→ {"id": "...", "email": "...", "name": "...", "role": "admin"}
```

### Server

```
GET  /api/servers                    → Liste aller Server
POST /api/servers                    → Server hinzufügen
GET  /api/servers/{id}/metrics       → Server-Metriken
```

### Websites

```
GET    /api/websites                 → Liste Websites
POST   /api/websites                 → Website erstellen
PUT    /api/websites/{id}            → Website bearbeiten
DELETE /api/websites/{id}            → Website löschen
POST   /api/websites/{id}/toggle     → Aktivieren/Deaktivieren
POST   /api/websites/{id}/ssl        → SSL aktivieren
```

### SSL

```
GET    /api/ssl                      → Liste Zertifikate
POST   /api/ssl                      → Zertifikat beantragen
POST   /api/ssl/{id}/renew           → Zertifikat verlängern
DELETE /api/ssl/{id}                 → Zertifikat löschen
```

### Datenbanken

```
GET    /api/databases                → Liste Datenbanken
POST   /api/databases                → Datenbank erstellen
DELETE /api/databases/{id}           → Datenbank löschen
GET    /api/databases/{id}/password  → Passwort abrufen
```

### DNS

```
GET    /api/dns/zones                → Liste Zonen
POST   /api/dns/zones                → Zone erstellen
GET    /api/dns/zones/{id}           → Zone + Records abrufen
DELETE /api/dns/zones/{id}           → Zone löschen
POST   /api/dns/zones/{id}/records   → Record hinzufügen
DELETE /api/dns/records/{id}         → Record löschen
```

### E-Mail

```
GET    /api/mail/domains             → Liste Mail-Domains
POST   /api/mail/domains             → Domain hinzufügen
DELETE /api/mail/domains/{id}        → Domain löschen
GET    /api/mail/accounts            → Liste Konten (?domain_id=...)
POST   /api/mail/accounts            → Konto erstellen
DELETE /api/mail/accounts/{id}       → Konto löschen
GET    /api/mail/aliases             → Liste Weiterleitungen
POST   /api/mail/aliases             → Weiterleitung erstellen
DELETE /api/mail/aliases/{id}        → Weiterleitung löschen
```

### Firewall

```
GET    /api/firewall                 → Liste Regeln (?server_id=...)
POST   /api/firewall                 → Regel erstellen
DELETE /api/firewall/{id}            → Regel löschen
POST   /api/firewall/{id}/toggle     → Aktivieren/Deaktivieren
POST   /api/firewall/reload          → Firewall neu laden
```

### Backups

```
GET    /api/backups/configs          → Liste Konfigurationen
POST   /api/backups/configs          → Konfiguration erstellen
DELETE /api/backups/configs/{id}     → Konfiguration löschen
POST   /api/backups/configs/{id}/run → Backup jetzt starten
GET    /api/backups/jobs             → Liste Jobs (?config_id=...)
```

### System

```
GET    /api/services                 → Liste Dienste (?server_id=...)
POST   /api/services/{name}/action   → Dienst-Aktion ({action: "start"})
GET    /api/crons                    → Liste Cron Jobs
POST   /api/crons                    → Cron Job erstellen
PUT    /api/crons/{id}               → Cron Job bearbeiten
DELETE /api/crons/{id}               → Cron Job löschen
GET    /api/logs                     → Liste verfügbare Logs
GET    /api/logs/{serverID}/{name}   → Log lesen (?lines=200)
GET    /api/files                    → Verzeichnis lesen (?server_id=&path=)
GET    /api/files/content            → Datei lesen (?server_id=&path=)
POST   /api/files/content            → Datei schreiben
DELETE /api/files                    → Datei/Ordner löschen
POST   /api/files/mkdir              → Ordner erstellen
GET    /api/packages/updates         → Verfügbare Updates
POST   /api/packages/update          → Updates installieren
```

### Updates & Einstellungen

```
GET    /api/system/info              → Panel-Version und System-Info
GET    /api/system/check-updates     → Auf Updates prüfen
POST   /api/system/update            → Update installieren
GET    /api/settings                 → Alle Einstellungen
PUT    /api/settings                 → Einstellung speichern {key, value}
GET    /api/users                    → Liste Benutzer
POST   /api/users                    → Benutzer erstellen
PUT    /api/users/{id}               → Benutzer bearbeiten
DELETE /api/users/{id}               → Benutzer löschen
POST   /api/users/{id}/password      → Passwort ändern
POST   /api/users/{id}/totp/setup    → 2FA einrichten
POST   /api/users/{id}/totp/verify   → 2FA aktivieren
DELETE /api/users/{id}/totp          → 2FA deaktivieren
```

---

## 24. Glossar

| Begriff | Erklärung |
|---|---|
| **Agent** | Leichtes Go-Programm, das auf jedem verwalteten Server läuft |
| **BIND** | Linux-DNS-Server (named) |
| **Certbot** | Let's Encrypt-Client für automatische SSL-Zertifikate |
| **CRON** | Zeitgesteuerter Task-Scheduler unter Linux |
| **DKIM** | DomainKeys Identified Mail — E-Mail-Signaturen |
| **DMARC** | Domain-based Message Authentication — E-Mail-Policy |
| **Document Root** | Wurzelverzeichnis einer Website |
| **Dovecot** | IMAP/POP3-Server für E-Mail-Abruf |
| **gRPC** | Protokoll für Service-zu-Service-Kommunikation (für zukünftige Versionen) |
| **JWT** | JSON Web Token — sicherer Authentifizierungstoken |
| **Let's Encrypt** | Kostenloser, automatischer SSL-Zertifikatsanbieter |
| **Master** | Hauptserver mit Web-UI und API |
| **mTLS** | Mutual TLS — gegenseitige Zertifikatsauthentifizierung |
| **Nginx** | Schneller Web-Server und Reverse Proxy |
| **PHP-FPM** | PHP FastCGI Process Manager |
| **Postfix** | Mail Transfer Agent (MTA) |
| **PostgreSQL** | Relationale Open-Source-Datenbank |
| **Redis** | In-Memory Key-Value-Store für Sessions/Cache |
| **SPF** | Sender Policy Framework — E-Mail-Authentifizierung |
| **SAN** | Subject Alternative Names — mehrere Domains im SSL-Zertifikat |
| **systemd** | Init-System und Service-Manager von Linux |
| **TOTP** | Time-based One-Time Password — Standard für 2FA |
| **UFW** | Uncomplicated Firewall — Frontend für iptables |
| **VHost** | Virtual Host — mehrere Domains auf einem Server |

---

*ControlPanelVPS ist Open Source (MIT Lizenz). Mitarbeit willkommen!*
*Repository: https://github.com/Sirbuschi2003/ControlPanelVPS-*
