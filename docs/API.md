# HomeLog — API Reference

Base URL: `/api/v1` (all endpoints require JWT authentication unless noted)

---

## Authentication

```http
POST /auth/register             # Register new user (no auth required)
POST /auth/login                # Login, returns JWT tokens (no auth required)
POST /auth/refresh              # Refresh access token
PUT  /settings/password         # Change password
```

## Properties

```http
GET    /properties              # List user properties
POST   /properties              # Create property
GET    /properties/:id          # Get property
PUT    /properties/:id          # Update property
DELETE /properties/:id          # Delete property
GET    /properties/:id/balance          # Get balance for property
GET    /properties/:id/balance/details  # Get detailed balance
GET    /properties/:id/settings         # Get household settings
PUT    /properties/:id/settings         # Update household settings
GET    /properties/:id/members          # List household members
POST   /properties/:id/members          # Add household member
```

## Categories

```http
GET    /categories              # List categories (global + user custom)
POST   /categories              # Create category
GET    /categories/:id          # Get category with subcategories
PUT    /categories/:id          # Update category
DELETE /categories/:id          # Delete category
```

## Expenses

```http
GET    /expenses                # List expenses (supports filters: from, to, category_id, project_id)
POST   /expenses                # Create expense (supports split)
GET    /expenses/:id            # Get expense
PUT    /expenses/:id            # Update expense (restricted if settled)
DELETE /expenses/:id            # Delete expense
GET    /expenses/stats          # Expense statistics (trend, by_category, totals)
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
PUT    /utilities/:id/bills/:bid        # Update bill
PUT    /utilities/:id/bills/:bid/full   # Full bill update (with template data)
DELETE /utilities/:id/bills/:bid        # Delete bill
POST   /utilities/:id/bills/upload      # Upload bill PDF

# Comparison
GET    /utilities/:id/compare-readings  # Compare self vs supplier readings
POST   /utilities/contract/upload       # Upload contract PDF
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
POST   /settlements             # Create settlement (marks splits as settled)
GET    /settlements/:id         # Get settlement
DELETE /settlements/:id         # Delete settlement
```

## Export / Import

```http
GET    /export/all              # Export all data as JSON
GET    /export/expenses         # Export expenses only
GET    /export/utilities        # Export utilities only
GET    /export/projects         # Export projects only
POST   /import                  # Import JSON data
```
