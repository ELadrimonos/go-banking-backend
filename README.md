# Go Banking Backend

A secure and scalable **banking backend system** built with **Go (Golang)**. This project is open-source and designed for learning, extension, and real-world use. It uses **PostgreSQL** as the main database and integrates with a **locally hosted instance of the Frankfurter API** for real-time currency exchange rates.

## Features

- User registration and authentication
- Bank account creation and management
- Deposits, withdrawals, and balance tracking
- Money transfers between accounts
- Multi-currency support
- Real-time currency conversion (Frankfurter API)
- PostgreSQL for persistent storage
- RESTful API architecture
- Modular and clean folder structure

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go (Golang) |
| Database | PostgreSQL |
| Currency Rates | Frankfurter API (self-hosted) |
| API Style | REST |
| Dependency Mgmt | Go Modules |
| Environment Vars | `.env` file |

## Project Structure

```
go-banking-backend/
├── account/
│   └── account.go
├── auth/
│   ├── auth.go
│   ├── errors.go
│   ├── helpers.go
│   ├── logger.go
│   ├── ratelimiter.go
│   ├── responses.go
│   └── validation.go
├── currency/
│   └── currency.go
├── db/
│   └── init.sql
├── transactions/
│   ├── deposit.go
│   └── transaction.go
├── .env.template
├── .gitignore
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── LICENSE
├── main.go
└── README.md
```

## Setup & Installation

### 1. Clone the repository
```bash
git clone https://github.com/ELadrimonos/go-banking-backend.git
cd go-banking-backend
```

### 2. Configure environment variables
Create a `.env` file from the `.env.template` template:

```env
POSTGRES_USER=user
POSTGRES_PASSWORD=password
POSTGRES_DB=banking
```

### 3. Start the services
```bash
docker-compose up --build
```

## API Endpoints (Examples)

| Method | Endpoint | Description |
|---|---|---|
| POST | `/signup` | Register a new user |
| POST | `/login` | User login |
| POST | `/change-password` | Change user password |
| GET | `/accounts` | Get user accounts |
| POST | `/create-account` | Create a new bank account |
| POST | `/deposit` | Deposit money into an account |
| GET | `/convert?from=USD&to=EUR&amount=100` | Convert an amount from one currency to another |

## Currency Exchange Integration

The system communicates with a locally hosted Frankfurter API for currency conversion. Example request:

```
GET http://localhost:8081/latest?amount=100&from=USD&to=EUR
```

## License

This project is open-source under the **MIT License**.

## Contributing

1.  Fork the repository
2.  Create a feature branch (`git checkout -b feature-name`)
3.  Commit your changes
4.  Open a Pull Request
