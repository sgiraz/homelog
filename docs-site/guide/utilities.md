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

Bills for rent and mortgage services are filed under their own **Rent/Mortgage**
subcategory of Home, so they no longer hide inside the Utilities slice of the
dashboard.

### The original PDF

Attach the provider's PDF to a bill — when you add it, or later with **Replace**
— and open it again from the bill with **View**. Stored PDFs are private: only
signed-in members of your household can open them.

Storage is capped per household (500 MB by default, see
[Self-Hosting](./self-hosting#configuration)) and uploads are rate-limited, so a
burst of files is refused with a message instead of filling the disk.

## Consumption analysis

The **Analysis** tab compares **billed** consumption against **actual**
consumption between consecutive bills, so you can spot estimate errors or
unexpected spikes.

### Comparing your readings with the provider's

For each bill, HomeLog looks for the self-reading that belongs to it: one taken
**inside the bill's period** or, failing that, the **closest one within the
reading window** (15 days by default). It compares the two and marks the bill as
OK, a warning or an anomaly.

The tolerance is a **base threshold** plus an extra amount **per day** between
the two readings, so a reading taken a week early is judged more leniently than
one taken the same day. Both thresholds, and the reading window, can be changed
in **Analysis → Comparison threshold settings**. If you read the meter rarely,
widen the window (up to 365 days).

When a bill shows **No data**, the message tells you why:

- no self-reading has been recorded for the service, or
- the closest one is outside the reading window. The message gives its date and
  how far it is from the bill: add a reading closer to that period, or widen the
  window.

A reading too far from a bill is deliberately not used — it would say nothing
about that period.

## Domiciliation & instalments

Two independent flags describe how a service is paid:

- **Domiciled** — paid automatically by direct debit.
- **Instalment-based** — billed in instalments.

A service can be either, both, or neither.

See [PDF Bill Templates](./pdf-templates) to automate reading figures straight
from your provider's PDFs.
