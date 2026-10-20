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
2. **Indica i campi.** Trascina il valore che vuoi su ogni campo (importo totale,
   periodo, lettura del fornitore…). HomeLog trova come ritrovarlo — dall'etichetta
   accanto, oppure dalla colonna della sua tabella e dalla riga — e mostra la regola
   con il valore che rilegge da quella stessa bolletta. Un valore che nessuna regola
   rilegge resta rosso: prendilo da dove sta accanto alla sua etichetta o nella sua
   tabella. Per il tipo lettura, tocca le parole che indicano una lettura stimata.
3. **Controlla e salva.** Se vuoi, prova il template su una seconda bolletta dello
   stesso fornitore, poi salvalo. Assegnalo come predefinito per un servizio, e le
   bollette future dello stesso fornitore vengono analizzate automaticamente.

I campi disponibili dipendono dal servizio. Quelli legati al contatore (la
lettura con cui si chiude la bolletta, la sua data e il tipo, la matricola)
compaiono solo per i servizi a consumo, come la **fine periodo**: su un servizio
a costo fisso viene derivata dalla frequenza di fatturazione configurata sul
servizio, quindi verifica che la frequenza corrisponda al ciclo reale del
fornitore.

HomeLog non tira mai a indovinare. Senza un template il PDF viene allegato ma non
si compila nulla, e un campo mappato dal template che una bolletta non riporta
resta vuoto ed è elencato come **non trovato**, così puoi inserirlo a mano.
Quando un fornitore cambia impaginazione, crea un nuovo template da una delle
nuove bollette.

## Condivisione e trasloco

Un template appartiene all'abitazione, non a chi l'ha creato: ogni membro del
nucleo può usarlo, modificarlo ed eliminarlo. Quando cambi casa, esporta un
template da **Template Estrazione** (un piccolo file JSON con le sue regole) e
importalo nella nuova abitazione.

## Strategia di estrazione

Per restare accurato anche quando un'etichetta compare più volte nella pagina,
HomeLog prova diverse strategie in ordine: prima gli ID globali univoci, poi la
posizione dell'etichetta di riferimento, poi il punto che hai indicato in origine,
infine una corrispondenza per pattern con il contesto circostante.

::: tip
I pattern di estrazione usano una sintassi regex senza look-ahead / look-behind,
che mantiene il parsing veloce e prevedibile anche su hardware poco potente.
:::
