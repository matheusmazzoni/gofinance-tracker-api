# GoFinance Tracker API 📊

Welcome to the GoFinance Tracker API\! This is a robust, production-ready backend service designed to help you manage your personal finances with clarity and control. Built with Go, it provides a clean, secure, and efficient foundation for any financial tracking application.

This project was built from the ground up, focusing on best practices like layered architecture, dependency injection, comprehensive testing, and production-readiness.

[](https://www.google.com/search?q=https://goreportcard.com/report/github.com/matheusmazzoni/gofinance-tracker-api)
[](https://www.google.com/search?q=https://github.com/matheusmazzoni/gofinance-tracker-api/actions)
[](https://opensource.org/licenses/MIT)

## ✨ Features

  * **🔐 Secure JWT Authentication:** Full user registration and login flow using JSON Web Tokens.
  * **🏦 Full CRUD for Core Entities:** Manage Accounts, Categories, Transactions, and Budgets.
  * **💰 Real-time Balance Calculation:** Account balances are calculated on-the-fly, accurately reflecting all incomes, expenses, and transfers.
  * **💸 Smart Budgeting:** Set monthly budgets per category and track your spending against them in real-time.
  * **🚀 Advanced Filtering:** A powerful `GET /transactions` endpoint that allows filtering by date range, description, type, amount, and more.
  * **⚙️ Production-Ready Architecture:**
      * Clean, layered architecture (Handlers, Services, Repositories).
      * Dependency Injection for easy testing and maintenance.
      * Structured, request-scoped logging with `zerolog`.
      * Graceful shutdown to prevent data loss.
      * Secure external configuration using environment variables.
      * Comprehensive test suite with both Unit and Integration tests.
  * **📚 Auto-generated API Documentation:** Interactive API documentation powered by Swagger (Swag).

## 🛠️ Tech Stack

| Component | Technology/Library |
| :--- | :--- |
| **Language** | Go (Golang) |
| **Web Framework** | Gin |
| **Database** | PostgreSQL |
| **DB Interaction** | `sqlx` |
| **Query Building**| `squirrel` |
| **Migrations** | `golang-migrate` |
| **Configuration** | `envdecode`, `.env` files |
| **Authentication** | `golang-jwt/jwt` |
| **Logging** | `zerolog` |
| **API Documentation** | `swaggo/swag` |
| **Testing** | `testify`, `testcontainers-go` |

## 🚀 Getting Started

Follow these steps to get the API running on your local machine for development and testing.

### Prerequisites

  * **Go:** Version 1.24 or later.
  * **Docker & Docker Compose:** Required to run the PostgreSQL database and for running integration tests.
  * **`migrate` CLI:** Install it to manage database migrations. [Installation guide](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate).

### 1\. Clone the Repository

```sh
git clone https://github.com/matheusmazzoni/gofinance-tracker-api.git
cd gofinance-tracker-api
```

### 2\. Install Dependencies

```sh
make tidy
```

### 3\. Set Up Configuration

The application is configured using environment variables. For local development, you can create a `.env` file in the project root.

1.  Copy the example file:

    ```sh
    cp .env.example .env
    ```

2.  **Edit the `.env` file** and fill in your details. The default values are configured to work with the Docker setup below.

    ```dotenv
    # .env

    # Application Environment: "development" or "production"
    APP_ENV="development"

    # Server Configuration
    SERVER_PORT="8080"

    # Database Connection URL
    DATABASE_URL="postgres://user:password@localhost:5432/finance_db?sslmode=disable"

    # JWT Secret Key (use a long, random string)
    JWT_SECRET_KEY="your-super-secret-and-long-jwt-key"
    ```

### 4\. Run the Database

We use Docker Compose to easily start a PostgreSQL instance.

```sh
docker-compose up -d
```

This will start a PostgreSQL server in the background on port 5432.

### 5\. Run Database Migrations

With the database running, apply the schema:

```sh
migrate -database ${DATABASE_URL} -path db/migrations up
```

*(You might need to export the `DATABASE_URL` from your `.env` file first with `export $(cat .env | xargs)`) or replace the variable with the actual URL.*

## 🏃‍♀️ Running the Application

### For Production (or Manually)

Build the binary and run it. The app will be configured from the environment variables.

```sh
make run
```

## 🧪 Running Tests

This project has a comprehensive test suite, including unit tests and integration tests that use `testcontainers-go`.

**Ensure Docker is running before executing tests.**

To run all tests for all packages:

```sh
make test
```

## 📚 API Documentation

Once the server is running, the interactive Swagger UI documentation is available at:

[http://localhost:8080/swagger/index.html](https://www.google.com/search?q=http://localhost:8080/swagger/index.html)

The documentation is automatically generated from the source code comments. If you change any annotations, remember to regenerate the docs by running:

```sh
make swag-docs
```

## 📁 Project Structure

```
.
├── cmd/api/            # Main application entrypoint
├── db/migrations/      # Database migration files (.sql)
├── api/                # Auto-generated Swagger files
└── internal/
    ├── api/            # Handles API concerns (DTOs, Handlers, Middlewares, Response helpers)
    ├── config/         # Configuration loading
    ├── db/             # Database utility functions (migrations runner)
    ├── logger/         # Logger setup
    ├── model/          # Core domain models (structs mirroring DB tables)
    ├── repository/     # Data access layer (interacts directly with the DB)
    ├── server/         # Server setup, dependency injection, and routing
    ├── service/        # Business logic layer
    └── testhelper/     # Shared utilities for integration tests
```

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](https://www.google.com/search?q=LICENSE) file for details.