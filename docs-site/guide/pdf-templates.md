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
2. **Tag the fields.** In a 3-step wizard, point at the value you want for each
   field (total amount, period, provider reading…). HomeLog records where it sits
   on the page and how to recognise it.
3. **Save the template.** Assign it as the default for a service, and future bills
   from the same provider are parsed automatically.

## Extraction strategy

To stay accurate even when a label appears more than once on a page, HomeLog
tries several strategies in order: unique global IDs first, then anchor/label
position, then the spot you originally tagged, then a pattern match with
surrounding context.

::: tip
Extraction patterns are restricted to a regex flavour without look-ahead /
look-behind, which keeps parsing fast and predictable on low-powered hardware.
:::
