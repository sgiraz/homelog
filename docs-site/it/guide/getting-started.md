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
| **Nucleo familiare** | Le persone che condividono le spese: membri, saldo e rimborsi appartengono al nucleo. Quasi tutti ne hanno uno. |
| **Proprietà** | Una casa di un nucleo, con le sue utenze e bollette. Un nucleo può averne più d'una. |
| **Membro** | Una persona del nucleo. I saldi sono tracciati tra i membri. |
| **Spesa** | Un costo, eventualmente diviso tra i membri e saldato nel tempo. |
| **Utenza** | Un servizio (luce, gas, acqua…) con letture e bollette. |
| **Bolletta** | Una fattura del fornitore, eventualmente collegata a una lettura del contatore. |
| **Progetto** | Un budget per un'attività una tantum — una ristrutturazione, un viaggio, un evento. |

Le abitazioni si gestiscono in **Impostazioni → Proprietà**, dove chi le ha
create può aggiungerle, modificarle, archiviarle o eliminarle. Un account fa
parte di un solo nucleo, e una nuova abitazione entra in quello: solo gli
admin del nucleo aggiungono abitazioni. Ogni nucleo ha un'abitazione
principale, quella proposta per le sue spese Casa.

Quando traslochi, **archivia** l'abitazione invece di eliminarla: lo storico
resta, le sue utenze ricevono ancora le ultime bollette e non compare più
quando aggiungi una spesa. La principale non si può archiviare: prima rendi
principale un'altra abitazione. Un'abitazione archiviata si può ripristinare in
qualsiasi momento.

Un'abitazione con spese Casa si può solo archiviare finché il nucleo ha altre
abitazioni: quelle spese fanno parte del bilancio del nucleo. Eliminare
un'abitazione senza spese elimina anche le sue utenze, bollette, letture e
progetti. Un'abitazione condivisa con altri account non si può eliminare,
perché contiene anche la loro storia. L'unica abitazione non si può eliminare.

## Dove andare ora

- [Self-hosting →](./self-hosting) — metti in piedi la tua istanza.
- [Spese e divisione →](./expenses) — le basi di tutti i giorni.
- [Utenze e bollette →](./utilities) — letture, bollette e consumi.
