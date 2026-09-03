---
title: Informativa privacy
description: Cosa memorizzano i siti pubblici di HomeLog, chi può vederlo e come esercitare i tuoi diritti.
---

# Informativa privacy

Questa informativa riguarda i tre siti pubblici di HomeLog:

| Sito | Cos'è |
| --- | --- |
| `homelog.dev` | Pagina di presentazione statica |
| `docs.homelog.dev` | Questo sito di documentazione |
| `demo.homelog.dev` | Demo pubblica, su un unico account condiviso |

**Non** riguarda un'istanza di HomeLog installata su un tuo server: in quel caso
il titolare del trattamento sei tu — vedi [Installazioni self-hosted](#installazioni-self-hosted).

**Ultimo aggiornamento:** 30 agosto 2026

## Chi è il titolare

**Titolare del trattamento:** Simone Girardi
**Contatto:** [privacy@homelog.dev](mailto:privacy@homelog.dev)

Scrivi a quell'indirizzo per qualsiasi cosa riguardi i tuoi dati personali:
arriva a una casella privata. Per tutto il resto (bug, richieste di
funzionalità, domande sul software) usa invece le
[issue su GitHub](https://github.com/sgiraz/homelog/issues), così la
discussione resta pubblica e utile anche ad altri.

## In breve

- Nessuna analytics, nessuna pubblicità, nessun tracciamento, nessuna profilazione.
- Nessun cookie che richieda consenso — per questo non vedi alcun banner.
- I siti salvano nel browser alcuni valori tecnici (lingua, tema e, sulla demo,
  un token di accesso). Non lasciano mai il tuo dispositivo.
- I web font sono serviti dai nostri domini: nessun fornitore di font ti vede.

## Cosa viene salvato nel tuo browser

Nulla di quanto segue è un cookie: si tratta di `localStorage` /
`sessionStorage`, che resta sul tuo dispositivo e non ci viene mai trasmesso. È
tutto o strettamente necessario o una preferenza che hai impostato tu, quindi ai
sensi dell'art. 5(3) della Direttiva ePrivacy (in Italia art. 122 del Codice
Privacy) è esente da consenso.

| Sito | Chiave | Finalità | Conservazione |
| --- | --- | --- | --- |
| `homelog.dev` | `homelog-lang` | La lingua che hai scelto | Finché non cancelli i dati del sito |
| `homelog.dev` | `homelog-theme` | Preferenza chiaro/scuro | Finché non cancelli i dati del sito |
| `docs.homelog.dev` | `vitepress-theme-appearance` | Preferenza chiaro/scuro | Finché non cancelli i dati del sito |
| `docs.homelog.dev` | `hl-lang-redirect` | Segna che il reindirizzamento una tantum alla documentazione italiana è già avvenuto | Fino alla chiusura della scheda |
| `demo.homelog.dev` | `token`, `refreshToken`, `user` | Ti mantiene autenticato sull'account demo | Fino al logout o alla cancellazione dei dati del sito |
| `demo.homelog.dev` | `themeMode`, `colorTheme`, `locale` | Preferenze di aspetto e lingua | Finché non cancelli i dati del sito |

Puoi cancellare tutto in qualsiasi momento dalla funzione "cancella dati del
sito" del browser. I siti continuano a funzionare: dimenticano solo le tue
preferenze.

## Log del server

Ogni sito è servito da un fornitore di hosting che registra i normali log di un
web server — indirizzo IP, data e ora, URL richiesto, user agent — per sicurezza
e diagnostica. La base giuridica è il legittimo interesse (art. 6(1)(f) GDPR) a
mantenere i servizi disponibili e sicuri. Conservazione e dettagli sono regolati
da ciascun fornitore:

- `homelog.dev` e `demo.homelog.dev` — [Render](https://render.com/privacy)
- `docs.homelog.dev` — [GitHub Pages](https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement)

## Terze parti contattate dal tuo browser

Caricare una pagina significa che il browser si collega a chi ne serve i
contenuti, e queste parti vedono necessariamente il tuo indirizzo IP. Su questi
siti si tratta di:

- **Caricando `homelog.dev`:** il CDN di Tailwind CSS (`cdn.tailwindcss.com`),
  che serve il motore di fogli di stile con cui la pagina è costruita.
- **Caricando `docs.homelog.dev` e `demo.homelog.dev`:** nulla oltre al
  fornitore di hosting.

I web font venivano caricati da Google Fonts sui primi due siti. Ora sono
serviti dai nostri domini, quindi Google non viene più contattato.

I link a GitHub, Ko-fi e GitHub Sponsors sono normali link in uscita: non viene
inviato loro nulla se non li clicchi.

## La demo pubblica

`demo.homelog.dev` gira su **un unico account condiviso** in cui entrano tutti i
visitatori. Le conseguenze vanno dette chiaramente:

- Tutto ciò che inserisci è **visibile a ogni altro visitatore** finché resta lì.
- L'intero dataset viene **cancellato e ricreato ogni ora**, e chiunque può
  farne il reset in qualsiasi momento. Nulla di ciò che inserisci viene conservato.
- La demo serve a mostrare il software. **Non inserire dati personali reali**,
  né tuoi né di altri.

Le credenziali dell'account demo sono pubblicate nella schermata di accesso:
l'accesso non è quindi una barriera di sicurezza.

## Installazioni self-hosted

Se installi HomeLog su un tuo server, i dati restano lì. A noi non arriva nulla:
non c'è telemetria, non c'è alcun "phone-home", non c'è raccolta di statistiche
d'uso. L'unica richiesta in uscita che l'applicazione fa per conto proprio è una
chiamata alle API delle release di GitHub per verificare se esiste una versione
più recente, effettuata dal *tuo* server e non dal tuo browser.

In quello scenario il titolare del trattamento dei dati domestici è chi gestisce
l'istanza, e questa informativa non si applica.

## I tuoi diritti

Ai sensi degli artt. 15–22 GDPR puoi chiedere l'accesso ai tuoi dati personali,
la rettifica o la cancellazione, la limitazione del trattamento, la portabilità,
e puoi opporti al trattamento basato sul legittimo interesse. Dato quanto poco
questi siti trattano, in pratica la questione riguarda i log del server descritti
sopra.

Scrivi al recapito indicato in cima a questa pagina. Hai inoltre diritto di
proporre reclamo a un'autorità di controllo — in Italia il
[Garante per la protezione dei dati personali](https://www.garanteprivacy.it).

## Modifiche

La data in cima alla pagina indica l'ultima modifica sostanziale. Lo storico
completo delle revisioni di questo documento è pubblico nella
[repository di HomeLog](https://github.com/sgiraz/homelog/commits/main/docs-site/it/privacy.md).
