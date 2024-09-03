# Transaction Processing Application

## Overview

This application processes incoming requests from 3rd-party providers, managing user account balances based on win/loss states.

## Prerequisites

- Go 1.20 or later
- Docker and Docker Compose
- PostgreSQL 16 or later (if running without Docker)

## Prerequisites

- Go 1.20 or later
- Docker and Docker Compose
- PostgreSQL 16 or later (if running without Docker)
- Migrate CLI tool (for database migrations if running without Docker)

## Quick Start

1. Clone the repository:
   ```sh
   git clone https://github.com/blackcloro/transaction-processor.git
   cd transaction-processor
   ```

2. Set up environment variables:
   ```sh
   cp .env.example .env
   # Edit .env with your desired configuration
   ```

3. Run with Docker:
   ```sh
   docker-compose up --build
   ```

   Or run locally:
   ```sh
   go run cmd/api/main.go
   ```


## API Endpoints

### Submit a Transaction

- **URL**: `/api/v1/transactions`
- **Method**: `POST`
- **Headers**:
   - `Content-Type: application/json`
   - `Source-Type: [game|server|payment]`
- **Body**:
  ```json
  {
    "state": "[win|lost]",
    "amount": "10.15",
    "transactionId": "unique-transaction-id"
  }
  ```

#### Example Request:
```http
POST /api/v1/transactions HTTP/1.1
Host: 127.0.0.1:4000
Source-Type: game
Content-Type: application/json
Content-Length: 76

{
  "state": "win",
  "amount": "10.15",
  "transactionId": "abc123-unique-transaction-id"
}
```

#### Notes:
- The `Source-Type` header can be one of: `game`, `server`, or `payment`.
- The `state` field in the body can be either `win` or `lost`.
- `win` state increases the user's balance, while `lost` state decreases it.
- Each `transactionId` is processed only once to prevent duplicate transactions.
- The account balance cannot go below zero.

### Check Server Health

- **URL**: `/api/v1/livez`
- **Method**: `GET`


## Development

### Running Tests

Execute the test suite:
```sh
go test ./... -v
```

### Database Migrations

Migrations are automatically applied when using Docker. For manual migration:

The migrate tool is used for database migrations. To install it, follow the instructions in the official GitHub repository.
[Detailed installation guide](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate#installation).

```sh
migrate -path migrations -database "postgres://<user>:<password>@<host>:<port>/<dbname>?sslmode=disable" up
```