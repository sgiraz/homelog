# PDF Bill Templates

::: warning Work in progress
This page is a stub. A full walkthrough with screenshots is coming.
:::

Provider bills are PDFs, and typing their figures by hand is tedious. HomeLog's
**template wizard** lets you teach the app to read a given provider's layout
**once** — after that, uploading a bill auto-fills the amount, period, meter
reading, and more.

## How it works

1. **Upload a sample bill.** HomeLog converts the PDF to text and detects tokens
   like amounts, dates, meter codes (POD/PDR), and numbers.
2. **Tag the fields.** Drag the value you want onto each field (total amount,
   period, provider reading…). HomeLog works out how to find it again — by the
   label next to it, or by the column of its table and which row — and shows the
   rule with the value it reads back from that same bill. A value no rule can
   read back stays red: take it from where it sits by its label or in its table.
   For the reading type, tap the words that mean an estimated reading.
3. **Check and save.** Optionally try the template on a second bill of the same
   provider, then save it. Assign it as the default for a service, and future
   bills from the same provider are parsed automatically.

The fields on offer depend on the service. Meter-specific ones (the reading the
bill closes on, its date and type, the meter serial number) appear only for
metered services, and so does **period end**: on a fixed-cost service it is
derived from the billing frequency configured on the service, so check that the
frequency matches the provider's real cycle.

HomeLog never guesses. Without a template a PDF is attached but nothing is
filled in, and a field the template maps but a bill does not show is left empty
and listed as **not found**, so you can type it in. When a provider changes its
layout, create a new template from one of the new bills.

## Sharing and moving house

A template belongs to the home, not to whoever created it: every member of the
household can use, edit and delete it. When you move, export a template from
**Extraction Templates** (a small JSON file with its rules) and import it in the
new home.

## Extraction strategy

To stay accurate even when a label appears more than once on a page, HomeLog
tries several strategies in order: unique global IDs first, then anchor/label
position, then the spot you originally tagged, then a pattern match with
surrounding context.

::: tip
Extraction patterns are restricted to a regex flavour without look-ahead /
look-behind, which keeps parsing fast and predictable on low-powered hardware.
:::
