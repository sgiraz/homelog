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

## Configurazione

| Variabile | Default | Descrizione |
| --- | --- | --- |
| `JWT_SECRET` | _(obbligatoria)_ | Segreto usato per firmare i token di autenticazione. |
| `DB_PATH` | `./data/homelog.db` | Posizione del database SQLite. |
| `GIN_MODE` | `debug` | Imposta `release` in produzione. |

## Approfondire

La [DEPLOY-GUIDE.md](https://github.com/sgiraz/homelog/blob/main/DEPLOY-GUIDE.md)
nel repository copre l'installazione su Raspberry Pi, l'accesso remoto con
Tailscale, i backup e la messa in sicurezza.
