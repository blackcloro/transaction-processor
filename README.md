# Transaction Processing Application

## Overview

This application processes incoming requests from 3rd-party providers, managing user account balances based on win/loss states.

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

### Configuration
The application is configured using environment variables. See .env.example for all available options. Key configurations:

```
TRANSACTION_PROCESSOR_PORT: Server port (default: 4000)
TRANSACTION_PROCESSOR_DB_DSN: Database connection string
TRANSACTION_PROCESSOR_DB_PASSWORD: Database password (used in docker-compose.yml)
TRANSACTION_PROCESSOR_WORKER_INTERVAL: Interval for post-processing worker
```

## Database Inspection

When running the application with Docker Compose, you may want to inspect the database directly. Here's how you can do that:

1. Ensure your Docker containers are running:
   ```sh
   docker-compose up -d
   ```

2. Connect to the PostgreSQL database:
   ```sh
   docker-compose exec db psql -U transactions -d transactions
   ```
   When prompted for a password, enter the `DB_PASSWORD` value from your `.env` file.

3. Once connected, you can run SQL queries to inspect the data. Here are some useful queries:

    - Check canceled odd records:
      ```sql
      SELECT id, transaction_id, account_id, source_type, state, amount, is_canceled, processed_at
      FROM transactions
      WHERE id % 2 = 1 AND is_canceled = true
      ORDER BY processed_at DESC
      LIMIT 10;
      ```

    - Check the current account balance:
      ```sql
      SELECT * FROM account WHERE id = 1;
      ```

    - View the most recent transactions:
      ```sql
      SELECT * FROM transactions ORDER BY processed_at DESC LIMIT 10;
      ```

4. To exit the PostgreSQL prompt, type:
   ```
   \q
   ```

Remember to exit the psql prompt when you're done. If you're finished with your Docker environment, you can shut it down with:
```sh
docker-compose down
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
