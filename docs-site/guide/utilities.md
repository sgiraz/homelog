# Utilities & Bills

::: warning Work in progress
This page is a stub. Screenshots and step-by-step walkthroughs are coming.
:::

A **utility** is a service tied to your home — electricity, gas, water, and so
on. Each utility keeps its own bills, and — when it has a meter — its readings
and consumption history.

## Metered vs fixed-cost services

The service type decides how HomeLog treats it:

- **Metered** — electricity, gas and water. These get the **Readings** and
  **Analysis** tabs, and consumption is tracked between bills.
- **Fixed-cost** — waste (TARI is billed on surface area, not on consumption),
  internet, insurance, rent and mortgage. Instead of readings they get a **Price
  history** tab, which records every change in the amount from one bill to the
  next.

## Meter readings

Log readings over time from the utility's **Readings** tab. HomeLog supports:

- single-value meters (gas, water),
- multi-band electricity meters (F1 / F2 / F3),
- estimated readings, when a bill is based on an estimate rather than a real one.

## Bills

Store each provider invoice under **Bills** — amount, period, due date, and the
provider's meter reading. A bill can be **linked to one of your readings**, which
is what powers the consumption analysis.

When you mark a bill as **paid**, HomeLog can automatically create the matching
expense (and split it), using the payer configured for that service.

## Consumption analysis

The **Analysis** tab compares **billed** consumption against **actual**
consumption between consecutive bills, so you can spot estimate errors or
unexpected spikes.

## Domiciliation & instalments

Two independent flags describe how a service is paid:

- **Domiciled** — paid automatically by direct debit.
- **Instalment-based** — billed in instalments.

A service can be either, both, or neither.

See [PDF Bill Templates](./pdf-templates) to automate reading figures straight
from your provider's PDFs.
