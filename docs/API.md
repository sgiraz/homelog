# HomeLog — API Reference

Base URL: `/api/v1` (all endpoints require JWT authentication unless noted)

---

## Version

```http
GET /version                    # Current version + update_available flag (public, no auth)
```

## Authentication

```http
POST /auth/register             # Register new user (no auth required)
POST /auth/login                # Login, returns JWT tokens (no auth required)
POST /auth/refresh              # Refresh access token
POST /auth/forgot-password      # Request password reset email (no auth required)
POST /auth/reset-password       # Complete password reset (no auth required)
PUT  /settings/password         # Change password (authenticated)
```

## Exchange Rates

```http
GET /exchange-rate              # Live currency exchange rate
```

## Properties

```http
GET    /properties              # List user properties
POST   /properties              # Create property
GET    /properties/:id          # Get property
PUT    /properties/:id          # Update property
DELETE /properties/:id          # Delete property
GET    /properties/:id/balance          # Get balance for property (long-term debts excluded)
GET    /properties/:id/balance/details  # Get detailed balance
GET    /properties/:id/debts            # Long-term debts: remaining, progress, payments
                                        # (filter: other_member_id)
GET    /properties/:id/settings         # Get household settings
PUT    /properties/:id/settings         # Update household settings
GET    /properties/:id/members          # List household members
POST   /properties/:id/members          # Add household member
```

## Join Requests

```http
POST   /join-requests                   # Request to join a property
GET    /join-requests                   # List pending join requests
PATCH  /join-requests/:id               # Resolve (approve/reject) a join request
GET    /properties/joinable             # List properties open to join requests
```

## Categories

```http
GET    /categories              # List categories (global + user custom)
POST   /categories              # Create category
GET    /categories/:id          # Get category with subcategories
PUT    /categories/:id          # Update category
DELETE /categories/:id          # Delete category
POST   /categories/:id/subcategories            # Add subcategory
DELETE /categories/:id/subcategories/:subId     # Remove subcategory
```

## Expenses

```http
GET    /expenses                # List expenses (filters: from, to, category_id, project_id)
POST   /expenses                # Create expense (supports split)
GET    /expenses/:id            # Get expense
PUT    /expenses/:id            # Update expense (restricted if settled)
DELETE /expenses/:id            # Delete expense
GET    /expenses/stats          # Expense statistics (trend, by_category, totals)
PATCH  /expenses/:id/long-term-debt  # Move the expense's shares out of the running
                                     # balance and into the debts ledger, or back
                                     # (409 once any share is partly settled)
```

## Expense Templates

```http
GET    /expense-templates       # List recurring expense templates
POST   /expense-templates       # Create template
PUT    /expense-templates/:id   # Update template
DELETE /expense-templates/:id   # Delete template
```

## Utilities

```http
GET    /utilities               # List utilities
POST   /utilities               # Create utility
GET    /utilities/:id           # Get utility details
PUT    /utilities/:id           # Update utility
DELETE /utilities/:id           # Delete utility

# Meter readings
POST   /utilities/:id/readings          # Add reading
GET    /utilities/:id/readings          # List readings
PUT    /utilities/:id/readings/:rid     # Update reading
DELETE /utilities/:id/readings/:rid     # Delete reading

# Bills
POST   /utilities/:id/bills             # Add bill
GET    /utilities/:id/bills             # List bills
PUT    /utilities/:id/bills/:bid        # Update bill (basic fields)
PUT    /utilities/:id/bills/:bid/full   # Full bill update (with template extraction data)
DELETE /utilities/:id/bills/:bid        # Delete bill
POST   /utilities/:id/bills/upload      # Upload bill PDF
PATCH  /utilities/:id/bills/:bid/installments/:instId  # Update installment (pay/unpay)

# Comparison
GET    /utilities/:id/compare-readings  # Compare self vs supplier readings

# Per-utility communications (bollettini, avvisi fornitore)
GET    /utilities/:id/communications
POST   /utilities/:id/communications
PUT    /utilities/:id/communications/:commId/read
DELETE /utilities/:id/communications/:commId

# Contract upload
POST   /utilities/contract/upload       # Upload contract PDF
```

## Communications (global — across all utilities)

```http
GET    /communications               # All communications
GET    /communications/unread-count
DELETE /communications/read          # Delete all read communications
```

## Notifications

```http
GET    /notifications
GET    /notifications/unread-count
PATCH  /notifications/:id/read
POST   /notifications/read-all
DELETE /notifications/:id
DELETE /notifications/read           # Delete all read notifications
```

## Bill Templates

```http
GET    /templates/bills         # List bill extraction templates
POST   /templates/bills         # Create template
PUT    /templates/bills/:id     # Update template
DELETE /templates/bills/:id     # Delete template
```

## PDF Processing

```http
POST   /pdf/extract-text        # Extract text from PDF (supports with_positions)
POST   /pdf/analyze             # Analyze PDF for template creation
DELETE /pdf/cleanup/:timestamp  # Cleanup temporary template images
```

## Projects

```http
GET    /projects                # List projects
POST   /projects                # Create project
GET    /projects/:id            # Get project with budget stats
PUT    /projects/:id            # Update project
DELETE /projects/:id            # Delete project
```

## Settings

```http
GET    /settings                # Get user settings
PUT    /settings                # Update user settings
POST   /settings/avatar         # Upload avatar (multipart)
DELETE /settings/avatar         # Remove avatar
GET    /settings/account/delete-check   # Pre-flight check before account deletion
POST   /settings/account/promote-admin  # Promote another member to admin
DELETE /settings/account        # Delete own account
```

## Members

```http
GET    /members/:id             # Get member
PUT    /members/:id             # Update member
DELETE /members/:id             # Delete member
```

## Settlements

```http
GET    /settlements             # List settlements (filter by property_id)
POST   /settlements             # Record a payment; allocated oldest-share-first,
                                # partial amounts allowed. Pass target_expense_id
                                # to pay down one long-term debt instead
POST   /settlements/compensate  # Offset a long-term debt against a credit the
                                # debtor holds on the other member (no cash moves)
GET    /settlements/:id         # Get settlement
DELETE /settlements/:id         # Delete settlement (reverses only its own allocations)
```

A settlement never marks a share settled by fiat: it records **allocations**
against individual shares, so an amount smaller than the outstanding balance
leaves the remainder owed. `POST` with an amount above what is outstanding
returns `400`.

## Admin

```http
DELETE /admin/users/:id         # Delete any user (admin only)
PUT    /admin/users/:id/role    # Toggle admin flag (admin only)
```

## Search

```http
GET /search                     # Full-text search (FTS5) across expenses, bills, utilities, projects
```

## Export / Import

```http
GET    /export/all              # Export all data as JSON
GET    /export/expenses         # Export expenses only
GET    /export/utilities        # Export utilities only
GET    /export/projects         # Export projects only
POST   /import                  # Import JSON data
```
