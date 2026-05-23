# Introduzione

::: warning Lavori in corso
Questa guida è in fase di scrittura. I contenuti sono incompleti e potrebbero cambiare.
:::

**HomeLog** è un gestionale self-hosted e multi-utente per il denaro che fa
funzionare una casa. Unisce tre cose che la maggior parte delle app tiene
separate:

- **Spese condivise** — dividi i costi tra i membri della famiglia e salda i conti.
- **Utenze** — registra le letture dei contatori e archivia le bollette di luce, gas e acqua.
- **Analisi** — confronta i consumi fatturati con quelli reali e scopri dove vanno i tuoi soldi.

Tutto gira in un unico container Docker, su hardware che controlli tu — da un VPS
a un Raspberry Pi 3B+.

## Prova prima di installare

Il modo più rapido per vedere cosa fa HomeLog è la **[demo live](https://demo.homelog.dev)**.
È precaricata con dati di esempio (che si resettano ogni ora), così puoi
esplorarla liberamente senza registrarti.

Quando sei pronto a usarlo sul serio, segui la guida al
[Self-hosting](./self-hosting) per installare la tua istanza privata.

## Concetti chiave

| Concetto | Cos'è |
| --- | --- |
| **Proprietà** | Una casa/nucleo familiare. Spese, utenze e membri appartengono a una proprietà. |
| **Membro** | Una persona che condivide le spese. I saldi sono tracciati per membro. |
| **Spesa** | Un costo, eventualmente diviso tra i membri e saldato nel tempo. |
| **Utenza** | Un servizio (luce, gas, acqua…) con letture e bollette. |
| **Bolletta** | Una fattura del fornitore, eventualmente collegata a una lettura del contatore. |
| **Progetto** | Un budget per un'attività una tantum — una ristrutturazione, un viaggio, un evento. |

## Dove andare ora

- [Self-hosting →](./self-hosting) — metti in piedi la tua istanza.
- [Spese e divisione →](./expenses) — le basi di tutti i giorni.
- [Utenze e bollette →](./utilities) — letture, bollette e consumi.
