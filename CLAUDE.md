# Claude – Coding Instructions

## Arbeitsweise & Token-Effizienz
- Antworten präzise und kurz halten. Kein Smalltalk, keine Wiederholungen.
- Code nur erklären wenn explizit danach gefragt wird.
- Nur geänderte Code-Blöcke zeigen, nicht die gesamte Datei.
- Unveränderte Bereiche mit `// ... rest unchanged` markieren.
- Keine Änderungen außerhalb des angefragten Scopes.
- Bei unklaren Anforderungen: genau 1 Rückfrage stellen, dann warten.
- Nur Dateien lesen, die für die aktuelle Aufgabe relevant sind.
- Keine automatischen Directory-Scans ohne explizite Aufforderung.

## Code-Qualität & Architektur
- Code auf höchstem Niveau: sauber, lesbar, wartbar.
- SOLID-Prinzipien einhalten (Single Responsibility, Open/Closed, etc.).
- DRY (Don't Repeat Yourself) – keine Code-Duplikate.
- Funktionen klein und fokussiert halten (max. eine Aufgabe pro Funktion).
- Sprechende Variablen- und Funktionsnamen – kein `tmp`, `x`, `data`.
- Konsistente Formatierung und Einrückung im gesamten Projekt.
- Fehlerbehandlung immer explizit implementieren, nie stillschweigend schlucken.
- Kommentare nur für das *Warum*, nicht das *Was* (Code soll selbsterklärend sein).

## Sicherheit (Security by Default)
- Niemals Secrets, API-Keys oder Passwörter im Code oder Logs.
- Alle Nutzereingaben validieren und sanitizen – kein blindes Vertrauen.
- SQL-Injection, XSS, CSRF-Schutz immer mitdenken.
- Principle of Least Privilege: minimale Berechtigungen, minimale Angriffsfläche.
- Abhängigkeiten (Dependencies) auf bekannte Schwachstellen hinweisen wenn erkannt.
- Keine veralteten oder unsicheren Bibliotheken vorschlagen.
- Authentifizierung und Autorisierung immer trennen.
- Sensible Daten in Transit und at Rest verschlüsseln.

## DSGVO-Konformität
- Keine personenbezogenen Daten (PII) ohne Rechtsgrundlage verarbeiten.
- Datensparsamkeit: nur Daten erheben, die wirklich benötigt werden.
- Löschkonzepte mitdenken (Retention Policies, Right to be forgotten).
- Logging: keine PII in Logs schreiben (Namen, E-Mails, IPs soweit möglich anonymisieren).
- Einwilligung (Consent) tracken wenn nötig – mit Timestamp und Versionierung.
- Datenübertragungen in Drittländer (außerhalb EU/EWR) explizit kennzeichnen.
- Bei neuen Features mit PII-Bezug: kurzen Hinweis auf DSGVO-Relevanz geben.

## NIS2-Konformität
- Sicherheitsrelevante Maßnahmen dokumentieren (Kommentare oder README).
- Incident-Response-Pfade mitdenken: Fehler müssen loggbar und nachvollziehbar sein.
- Zugriffskontrollen und Audit-Logs bei kritischen Operationen vorsehen.
- Supply-Chain-Sicherheit: auf Herkunft und Vertrauenswürdigkeit von Abhängigkeiten achten.
- Bei kritischer Infrastruktur oder sensiblen Systemen: explizit auf erhöhten Schutzbedarf hinweisen.

## Rechtliches (Allgemein)
- Keine Nutzung von Code oder Assets mit unklarer oder restriktiver Lizenz.
- Open-Source-Lizenzen beachten (GPL-Viralität, MIT vs. Apache 2.0 etc.).
- Bei KI-generierten Inhalten auf mögliche IP-Risiken hinweisen.

## Sprache
- Antworten auf Deutsch, Code-Kommentare auf Englisch.