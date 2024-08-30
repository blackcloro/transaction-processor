# Transaction Processing Application

## Overview

This application processes incoming requests from 3rd-party providers, managing user account balances based on win/loss states. It's built with Go and PostgreSQL, emphasizing concurrent processing, data consistency, and scalability.

## Prerequisites

- Go 1.20 or later
- Docker and Docker Compose
- PostgreSQL 13 or later (if running without Docker)

## Setup and Installation

1. Clone the repository:
   ```
   git clone https://github.com/blackcloro/transaction-processor.git
   cd transaction-processor
   ```

2. Run the app:
   ```
   go run main.go
   ```
    
## Testing

Run the test suite:
```
go test ./... -v
```
