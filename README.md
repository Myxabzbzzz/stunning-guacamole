# Billing API

A reliable RESTful API for managing user accounts, transactions, and billing operations. Built in Go, this API provides secure user authentication, transaction management, and real-time account balance tracking.

## Features

- **User Management**: Create, retrieve, and manage user accounts
- **Transaction Processing**: Handle money transfers between users
- **Authentication**: Secure session-based authentication
- **Balance Tracking**: Real-time account balance monitoring
- **Transaction History**: Full logging and retrieval of transactions
- **Soft Delete**: Safe user deletion with data retention
- **CORS Support**: Cross-origin resource sharing enabled
- **Request Logging**: Detailed request/response logging
- **Database Integration**: PostgreSQL with migration support

## Technology Stack

- **Language**: Go 1.23+
- **Framework**: Gorilla Mux (HTTP router)
- **Database**: PostgreSQL
- **Authentication**: Gorilla Sessions
- **Configuration**: TOML
- **Logging**: Logrus
- **Validation**: Ozzo Validation
- **Testing**: Testify

## Project Structure

```
billing_API/
├── cmd/apiserver/          # Application entry point
├── configs/                # Configuration files
├── internal/               # Private application code
│   ├── app/
│   │   ├── apiserver/      # HTTP server implementation
│   │   ├── model/          # Data models
│   │   └── store/          # Data access layer
│   │       ├── sqlstore/   # PostgreSQL implementation
│   │       └── teststore/  # Test storage implementation
├── migrations/             # Database migrations
├── go.mod                  # Go module file
├── go.sum                  # Go module checksums
└── Makefile                # Build automation
```

## Requirements

- Go 1.23 or higher
- PostgreSQL 12 or higher
- Make (optional, for build automation)

## Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd billing_API
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up the database**
   ```bash
   # Create the PostgreSQL database
   createdb restapi_dev
   
   # Run migrations
   psql -d restapi_dev -f migrations/20240702123000_init_schema.up.sql
   ```

4. **Configure the application**
   
   Edit `configs/apiserver.toml` with your database credentials:
   ```toml
   bind_addr = ":8080"
   log_level = "debug"
   database_url = "host=localhost dbname=restapi_dev password=your_password sslmode=disable"
   session_key = "your_session_key_here"
   ```

## Usage

### Build the application

```bash
# Using Make
make build

# Or directly with Go
go build -v ./cmd/apiserver
```

### Run the application

```bash
# Using the built binary
./apiserver

# Or with a custom config path
./apiserver -config-path=/path/to/config.toml

# Or directly with Go
go run ./cmd/apiserver
```

### Run tests

```bash
# Using Make
make test

# Or directly with Go
go test -v -race -timeout 30s ./...
```

## API Endpoints

### Authentication

- `POST /sessions` - User login
- `GET /private/whoami` - Get current user info (authenticated)

### User Management

- `POST /users` - Create a new user
- `GET /users` - List all users
- `GET /users/{id}` - Get user by ID
- `GET /users/{id}/balance` - Get user balance
- `GET /users/{id}/transactions` - Get user transaction history
- `POST /users/{id}/delete` - Soft delete user

### Transaction Management

- `POST /transactions` - Create a new transaction
- `GET /transactions` - List all transactions
- `GET /transactions/{id}` - Get transaction by ID
- `POST /transactions/{id}/cancel` - Cancel transaction

### System

- `GET /status` - API status check

## API Examples

### Create a user

```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Ivan Ivanov",
    "phone_number": "+998908620059",
    "card_number": "1234567890123456",
    "amount_of_money": 1000.00,
    "email": "ivan@example.com",
    "password": "securepassword"
  }'
```

### User login

```bash
curl -X POST http://localhost:8080/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "email": "ivan@example.com",
    "password": "securepassword"
  }'
```

### Create a transaction

```bash
curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "from_user_id": 1,
    "to_user_id": 2,
    "amount_of_money": 100.00
  }'
```

### Get user balance

```bash
curl http://localhost:8080/users/1/balance
```

## Database Schema

### Users Table
- `id` - Primary key
- `name` - Full name of the user
- `phone_number` - Contact phone number
- `card_number` - Unique card identifier
- `amount_of_money` - Account balance
- `email` - Unique email address
- `encrypted_password` - Hashed password
- `is_deleted` - Soft delete flag

### Transactions Table
- `id` - Primary key
- `from_user_id` - Sender user ID
- `to_user_id` - Recipient user ID
- `amount_of_money` - Transaction amount
- `transaction_time` - Timestamp
- `is_deleted` - Soft delete flag

## Configuration

The application uses TOML configuration files. Main configuration options:

- `bind_addr` - Server bind address (default: ":8080")
- `log_level` - Logging level (debug, info, warn, error)
- `database_url` - PostgreSQL connection string
- `session_key` - Secret key for session encryption

## Development

### Project Structure

The project follows clean architecture principles:

- **Models** (`internal/app/model/`) - Business logic and data structures
- **Store** (`internal/app/store/`) - Data access abstraction
- **API Server** (`internal/app/apiserver/`) - HTTP server and handlers
- **Configuration** (`configs/`) - Application configuration

### Adding New Features

1. Define models in `internal/app/model/`
2. Implement store methods in `internal/app/store/`
3. Add HTTP handlers in `internal/app/apiserver/server.go`
4. Update router configuration
5. Add tests for new functionality

## Testing

The project includes comprehensive tests:

- Unit tests for models and business logic
- Integration tests for the store layer
- HTTP handler tests
- End-to-end API tests

Run tests:
```bash
go test -v ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

[Add license information here]

## Support

For support and questions, please [create an issue](issue-link) or contact the development team.

## Swagger UI

Swagger UI is available at:

- http://localhost:8080/swagger/index.html?url=/swagger.yaml

## New Endpoints

- `POST /logout` — User logout (session removal)
- `POST /users/{id}/restore` — Restore user (undo soft-delete)

## Uniqueness Requirements

- Email, phone_number, and card_number must be unique for each user.

## Rate limiting

- By default, the limit is: no more than 10 requests per second from one IP.
- If the limit is exceeded, 429 Too Many Requests is returned.

## Security

- CORS is allowed only for http://localhost:3000 and https://your-domain.com
- Rate limiting is enabled for all endpoints

## Tests and Environment Variables

To run integration tests, the `DATABASE_URL` environment variable is required:

```bash
export DATABASE_URL="host=localhost dbname=restapi_test sslmode=disable"
go test -v ./...
```

## Swagger/OpenAPI

OpenAPI documentation is in `docs/swagger.yaml`. It can be viewed via Swagger UI (see above).
