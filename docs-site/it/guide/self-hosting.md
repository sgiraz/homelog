# Self-hosting

::: warning Lavori in corso
Questa guida è in fase di scrittura. Per il riferimento completo e aggiornato sul
deploy consulta la
[DEPLOY-GUIDE.md](https://github.com/sgiraz/homelog/blob/main/DEPLOY-GUIDE.md)
nel repository.
:::

HomeLog si distribuisce come **un singolo container Docker**: il binario Go serve
sia l'API sia il frontend incorporato, con SQLite come database. Nessun database
o web server separato da gestire.

## Avvio rapido con Docker

```bash
git clone https://github.com/sgiraz/homelog.git && cd homelog
cp .env.example .env

# Imposta un JWT secret sicuro (l'unica impostazione obbligatoria):
#   Linux/macOS:  openssl rand -base64 32
# Incolla il risultato in .env come JWT_SECRET=...

docker compose up -d
```

Poi apri **http://localhost:8080**, registra il primo account e inizia a tracciare.

## Usare l'immagine pre-buildata

```bash
mkdir homelog && cd homelog
echo "JWT_SECRET=$(openssl rand -base64 32)" > .env
curl -O https://raw.githubusercontent.com/sgiraz/homelog/main/docker-compose.yml
docker compose up -d
```

Le immagini multi-arch (amd64 / arm64 / arm/v7) sono pubblicate su
[Docker Hub](https://hub.docker.com/u/sgira), quindi lo stesso file compose
funziona anche su un Raspberry Pi.

## Scegliere la versione

Il file compose esegue `sgira/homelog:${HOMELOG_TAG:-latest}`, quindi per
impostazione predefinita ottieni la versione corrente senza configurare nulla.
Imposta `HOMELOG_TAG` nel tuo `.env` per eseguirne un'altra:

```bash
HOMELOG_TAG=2026.10.13   # fissa una versione esatta
HOMELOG_TAG=rc           # l'ultima release candidate
```

Poi `docker compose pull && docker compose up -d`.

Una release candidate è una build pubblicata prima di una release, perché possa
essere provata su un'installazione vera. Non viene annunciata come
aggiornamento e la tua istanza non te la proporrà mai. Eseguirla è una scelta
con un'avvertenza che vale la pena conoscere: lo schema del database viene
migrato all'avvio e non torna indietro, quindi passare poi da `rc` a `latest`
potrebbe non avviarsi. **Fai un backup di `./data` prima di cambiare** e resta
su `latest` se non intendi contribuire ai test.

## Aggiornare dalla 2026.09.11 o precedenti

Le versioni successive alla 2026.09.11 cambiano il modo in cui sono salvati i
template delle bollette e le loro letture. Un database della 2026.09.11 o
precedente va convertito una volta, prima che la nuova versione possa avviarsi.
Se salti questo passo, HomeLog non si avvia e il suo log rimanda qui.

```bash
docker compose stop homelog
cp -a data data-backup-$(date +%Y%m%d)    # prima il backup

docker compose pull
docker compose run --rm homelog ./upgrade
docker compose up -d
```

`./upgrade` converte il database ed esce. Assegna ogni template alla casa che
serve, trasforma la stima di una bolletta nella sua lettura di chiusura e toglie
il vecchio template predefinito per servizio. Poi la nuova versione si avvia e
migra il resto. Lanciarlo due volte non fa danni: la seconda volta non trova
niente da fare.

Se la tua istanza è più vecchia della 2026.09.11, aggiorna prima alla 2026.09.11
(`HOMELOG_TAG=2026.09.11`), avviala una volta, poi segui i passi qui sopra.

## Configurazione

| Variabile | Default | Descrizione |
| --- | --- | --- |
| `JWT_SECRET` | _(obbligatoria)_ | Segreto usato per firmare i token di autenticazione. |
| `DB_PATH` | `./data/homelog.db` | Posizione del database SQLite. |
| `UPLOAD_QUOTA_MB` | `500` | Dimensione massima dei PDF delle bollette conservati per casa, in MB. `0` toglie il limite. |
| `PORT` | `8080` | Porta su cui il server è in ascolto. |
| `GIN_MODE` | `debug` | Imposta `release` in produzione. |
| `TZ` | _(di sistema)_ | Fuso orario del container, ad es. `Europe/Rome`. |
| `CORS_ALLOWED_ORIGINS` | _(solo localhost)_ | Origini, separate da virgola, autorizzate a chiamare l'API da un altro sito. Raramente serve: il frontend è servito dallo stesso binario. |

Anche i caricamenti di PDF delle bollette sono limitati: 20 al minuto per utente.

## Approfondire

La [DEPLOY-GUIDE.md](https://github.com/sgiraz/homelog/blob/main/DEPLOY-GUIDE.md)
nel repository copre l'installazione su Raspberry Pi, l'accesso remoto con
Tailscale, i backup e la messa in sicurezza.
