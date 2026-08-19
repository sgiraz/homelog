# Template PDF delle bollette

::: warning Lavori in corso
Questa pagina è una bozza. Una guida completa con screenshot è in arrivo.
:::

Le bollette dei fornitori sono PDF, e digitarne i dati a mano è noioso. Il
**wizard dei template** di HomeLog ti permette di insegnare all'app a leggere il
layout di un dato fornitore **una volta sola** — dopodiché, caricando una
bolletta, importo, periodo, lettura del contatore e altro si compilano da soli.

## Come funziona

1. **Carica una bolletta di esempio.** HomeLog converte il PDF in testo e rileva
   token come importi, date, codici contatore (POD/PDR) e numeri.
2. **Indica i campi.** In un wizard a 3 step, punta al valore che vuoi per ogni
   campo (importo totale, periodo, lettura del fornitore…). HomeLog memorizza dove
   si trova nella pagina e come riconoscerlo.
3. **Salva il template.** Assegnalo come predefinito per un servizio, e le
   bollette future dello stesso fornitore vengono analizzate automaticamente.

I campi disponibili dipendono dal servizio. Quelli legati al contatore (lettura
del fornitore, stime) compaiono solo per gas e acqua, e la **fine periodo** solo
per i servizi a consumo: su un servizio a costo fisso viene derivata dalla
frequenza di fatturazione configurata sul servizio, quindi verifica che la
frequenza corrisponda al ciclo reale del fornitore.

## Strategia di estrazione

Per restare accurato anche quando un'etichetta compare più volte nella pagina,
HomeLog prova diverse strategie in ordine: prima gli ID globali univoci, poi la
posizione dell'etichetta di riferimento, poi il punto che hai indicato in origine,
infine una corrispondenza per pattern con il contesto circostante.

::: tip
I pattern di estrazione usano una sintassi regex senza look-ahead / look-behind,
che mantiene il parsing veloce e prevedibile anche su hardware poco potente.
:::
