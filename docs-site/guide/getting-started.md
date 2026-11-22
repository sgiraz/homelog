# Introduction

::: warning Work in progress
This guide is being written. Content is incomplete and may change.
:::

**HomeLog** is a self-hosted, multi-user manager for the money that keeps a home
running. It combines three things most apps keep separate:

- **Shared expenses** — split costs between household members and settle up.
- **Utilities** — log meter readings and store provider bills for electricity, gas and water.
- **Analysis** — see billed vs. actual consumption and where your money goes.

Everything runs in a single Docker container on hardware you control — from a VPS
to a Raspberry Pi 3B+.

## Try before you install

The fastest way to see what HomeLog does is the **[live demo](https://demo.homelog.dev)**.
It comes pre-loaded with sample data (which resets every hour), so you can click
around freely without signing up.

When you're ready to use it for real, follow the [Self-Hosting](./self-hosting)
guide to deploy your own private instance.

## Core concepts

| Concept | What it is |
| --- | --- |
| **Household** | The people who share expenses: members, balance and settlements belong to it. Most people have one. |
| **Property** | A home of a household, with its utilities and bills. A household can have several. |
| **Member** | A person of a household. Balances are tracked between members. |
| **Expense** | A cost, optionally split between members and settled over time. |
| **Utility** | A service (electricity, gas, water…) with readings and bills. |
| **Bill** | A provider invoice, optionally linked to a meter reading. |
| **Project** | A budget for a one-off effort — a renovation, a trip, an event. |

Homes are managed in **Settings → Properties**, where their owner can add,
edit, archive or delete them. An account belongs to one household, and a new
home joins it: only the household's admins add homes. Each household has one
primary home, the default for its Home expenses.

When you move out, **archive** the home instead of deleting it: its history
stays, its services still take the last bills, and it no longer shows up when
you add an expense. The primary home can't be archived: make another home
primary first. An archived home can be restored at any time.

A home with Home expenses can only be archived while its household has other
homes: those expenses are part of the household's balance. Deleting a home
without expenses also deletes its utilities, bills, readings and projects. A
home shared with other accounts can't be deleted, since it holds their history
too. Your only home can't be deleted.

## Where to next

- [Self-Hosting →](./self-hosting) — get your own instance running.
- [Expenses & Splitting →](./expenses) — the day-to-day basics.
- [Utilities & Bills →](./utilities) — readings, bills and consumption.
