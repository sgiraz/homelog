# Utenze e bollette

::: warning Lavori in corso
Questa pagina è una bozza. Screenshot e procedure passo-passo sono in arrivo.
:::

Un'**utenza** è un servizio legato alla tua casa — luce, gas, acqua e così via.
Ogni utenza tiene le proprie bollette e — se ha un contatore — le letture e lo
storico dei consumi.

## Servizi a consumo e servizi fissi

Il tipo di servizio decide come HomeLog lo tratta:

- **A consumo** — luce, gas e acqua. Hanno le schede **Letture** e **Analisi**, e
  il consumo viene tracciato tra una bolletta e l'altra.
- **A costo fisso** — rifiuti (la TARI si calcola sulla superficie, non sul
  consumo), internet, assicurazione, affitto e mutuo. Al posto delle letture
  hanno la scheda **Storico prezzi**, che registra ogni variazione di importo da
  una bolletta alla successiva.

## Letture del contatore

Registra le letture nel tempo dalla scheda **Letture** dell'utenza. HomeLog
supporta:

- contatori a valore singolo (gas, acqua),
- contatori elettrici multi-fascia (F1 / F2 / F3),
- letture stimate, quando una bolletta si basa su una stima invece che su una
  lettura reale.

## Bollette

Archivia ogni fattura del fornitore in **Bollette** — importo, periodo, scadenza
e lettura del contatore del fornitore. Una bolletta può essere **collegata a una
delle tue letture**, ed è ciò che alimenta l'analisi dei consumi.

Quando segni una bolletta come **pagata**, HomeLog può creare automaticamente la
spesa corrispondente (e dividerla), usando il pagatore configurato per quel
servizio.

## Analisi dei consumi

La scheda **Analisi** confronta il consumo **fatturato** con quello **effettivo**
tra bollette consecutive, così puoi individuare errori di stima o picchi
inattesi.

## Domiciliazione e rate

Due flag indipendenti descrivono come viene pagato un servizio:

- **Domiciliato** — pagato automaticamente con addebito diretto.
- **Rateizzato** — fatturato a rate.

Un servizio può essere l'uno, l'altro, entrambi o nessuno dei due.

Vedi [Template PDF delle bollette](./pdf-templates) per automatizzare la lettura
dei dati direttamente dai PDF del tuo fornitore.
