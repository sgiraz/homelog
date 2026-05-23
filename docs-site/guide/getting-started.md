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
| **Property** | A home/household. Expenses, utilities and members belong to a property. |
| **Member** | A person who shares expenses. Balances are tracked per member. |
| **Expense** | A cost, optionally split between members and settled over time. |
| **Utility** | A service (electricity, gas, water…) with readings and bills. |
| **Bill** | A provider invoice, optionally linked to a meter reading. |
| **Project** | A budget for a one-off effort — a renovation, a trip, an event. |

## Where to next

- [Self-Hosting →](./self-hosting) — get your own instance running.
- [Expenses & Splitting →](./expenses) — the day-to-day basics.
- [Utilities & Bills →](./utilities) — readings, bills and consumption.
