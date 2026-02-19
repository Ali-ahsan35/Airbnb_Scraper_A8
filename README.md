# Airbnb Scraper A8

GitHub Repository: `https://github.com/Ali-ahsan35/Airbnb_Scraper_A8`

## Overview

This project is a concurrent Airbnb scraper built with Go and `chromedp`.  
It:

- scrapes listing data from Airbnb section pages
- collects listing detail data concurrently using a worker pool
- cleans and deduplicates data
- saves output to CSV
- stores cleaned records in PostgreSQL
- prints an insights report in terminal

The scraper is designed for dynamic pages where static HTTP scraping is unreliable.  
It uses browser automation, concurrency controls, retry logic, and data-cleaning steps to create a practical end-to-end pipeline:

1. collect section URLs from Airbnb home/search entry points
2. collect property URLs from section pages (with next-page handling)
3. scrape detail pages concurrently with a worker pool
4. clean and deduplicate records
5. persist cleaned records to CSV and PostgreSQL
6. compute and print analytics from the cleaned dataset

## Features

- Dynamic Airbnb scraping with `chromedp`
- Concurrent detail-page scraping with configurable worker pool
- Stealth handling (rotating user-agent + browser fingerprint masking)
- Retry mechanism with exponential backoff
- Random delay and timeout-based request control
- Duplicate URL avoidance (thread-safe)
- Section pagination handling (page 1 + page 2 per section)
- Data cleaning and deduplication before insights/storage
- CSV export to `output/listings.csv`
- PostgreSQL schema creation and batch insert with conflict-safe URL dedupe
- Terminal insights report:
- total listings
- Airbnb listings
- average/min/max price
- most expensive property
- top 5 rated properties
- listing count by location

## Tech Stack

- Go `1.25.5` (from `go.mod`)
- `chromedp` for browser automation
- PostgreSQL (Docker Compose)
- `pgx/v5` for PostgreSQL access

## Project Structure

```text
airbnb-scraper/
├── main.go                         # Application entrypoint: orchestrates scrape -> clean -> save -> report
├── go.mod                          # Go module and dependency definitions
├── go.sum                          # Dependency checksums
├── docker-compose.yml              # Local PostgreSQL service (port 5433)
├── README.md                       # Project documentation
│
├── config/
│   └── config.go                   # Runtime configuration (scraping, retries, DB connection)
│
├── models/
│   └── listing.go                  # Core data structures: Listing, ScrapeJob, ScrapeResult
│
├── scraper/
│   └── airbnb/
│       ├── scraper.go              # chromedp scraping logic, selectors, parsing, URL dedupe
│       └── worker_pool.go          # Concurrent worker pool for detail-page scraping
│
├── services/
│   └── insights.go                 # Data cleaning + analytics report generation/printing
│
├── storage/
│   ├── csv_writer.go               # CSV export for scraped/cleaned listings
│   └── postgres_writer.go          # PostgreSQL schema setup + batch insert writer
│
├── utils/
│   ├── delay.go                    # Randomized request delay helper
│   ├── logger.go                   # Colored terminal logging helpers
│   ├── retry.go                    # Retry with exponential backoff
│   └── stealth.go                  # Browser stealth options for chromedp
│
└── output/
    └── listings.csv                # Generated CSV output (created after runs)
```

### Architecture Notes

- `main.go` keeps control flow simple and delegates each responsibility to a dedicated package.
- `scraper/airbnb` focuses only on data extraction and concurrency.
- `services/insights.go` handles business logic (cleaning and metrics), separate from scraping and persistence.
- `storage/` separates output targets (CSV and PostgreSQL) so you can swap/extend persistence cleanly.

## Prerequisites

- Go installed (compatible with `go.mod`)
- Docker and Docker Compose installed
- Internet access for scraping

## Quick Start

### 1. Clone repository

```bash
git clone https://github.com/Ali-ahsan35/Airbnb_Scraper_A8.git
cd Airbnb_Scraper_A8
```

### 2. Start PostgreSQL

```bash
docker compose up -d
docker compose ps
```

Expected container:

- `airbnb-scraper-postgres`
- port mapping: `5433 -> 5432`

### 3. Run scraper

```bash
go run main.go
```

This will:

- scrape Airbnb listings
- write CSV to `output/listings.csv`
- insert cleaned records into PostgreSQL database `airbnb_scraper`
- print insight tables in terminal

## Database Access and Verification

### Open PostgreSQL shell

```bash
docker exec -it airbnb-scraper-postgres psql -U postgres -d airbnb_scraper
```

### Useful checks

```sql
\dt
SELECT COUNT(*) FROM listings;
SELECT id, title, price, location, rating FROM listings ORDER BY id DESC LIMIT 20;
SELECT url, COUNT(*) FROM listings GROUP BY url HAVING COUNT(*) > 1;
```

### Delete all scraped rows

```sql
TRUNCATE TABLE listings;
```

## Configuration

Current runtime config is defined directly in `config/config.go` via `DefaultConfig()`:

- `MaxPages` (number of section links processed)
- `MaxWorkers` (concurrent detail workers)
- `RequestTimeout`
- `MinDelay` / `MaxDelay`
- `MaxRetries`
- DB settings (`DBHost`, `DBPort`, `DBUser`, `DBPassword`, `DBName`, `DBSSLMode`)

If you want more/less data:

- increase/decrease `MaxPages` in `config/config.go`
- adjust listing slice count in `scraper/airbnb/scraper.go` (`.slice(0, N)`)

## Docker Compose

`docker-compose.yml` provisions PostgreSQL with:

- DB name: `airbnb_scraper`
- user: `postgres`
- password: `postgres`
- host port: `5433`

## Troubleshooting

### PostgreSQL connection refused

Check:

```bash
docker compose ps
docker compose logs postgres --tail=100
```

Ensure `config/config.go` uses `DBPort: 5433`.

### Port already allocated

If another service is using port `5433`, update `docker-compose.yml` port mapping and `DBPort` in config to match.

### Fewer listings than expected

Expected max is a theoretical cap. Actual results can be lower due to:

- section/page timeouts
- missing "Next" button for some sections
- duplicate URL filtering
- failed listing detail pages

## Notes

- This implementation uses `chromedp` for dynamic Airbnb pages.
- `output/listings.csv` is generated output and may be ignored by Git.
