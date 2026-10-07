# Freelance Escrow Backend

Backend for a freelance-escrow marketplace, built with **Go** and **PostgreSQL**.

The project is designed to address trust and payment risks between clients and freelancers by providing structured account management, authentication, project workflows, and escrow-based payment flows.

> **Status:** In Progress

## Tech Stack

- **Go**
- **PostgreSQL**
- **`database/sql`**
- **pgx** — PostgreSQL driver
- **bcrypt** — password hashing
- **JWT** — authentication tokens
- **godotenv** — local environment configuration

## Current Features

- Client registration
- Freelancer registration
- Separate client and freelancer accounts
- Globally unique email identities
- Password hashing with bcrypt
- PostgreSQL database transactions
- Login with email and password
- JWT-based authentication
- Role information included in authenticated tokens
- Development database schema

## Project Structure

```text
freelance-backend/
├── db/
│   ├── auth.go
│   ├── client.go
│   ├── db.go
│   ├── errors.go
│   ├── freelancer.go
│   └── schema.sql
├── handlers/
│   ├── auth.go
│   ├── client.go
│   └── freelancer.go
├── go.mod
├── go.sum
├── main.go
└── .gitignore
```

### `db/`

Contains database connectivity, queries, account persistence, and the development schema.

### `handlers/`

Contains HTTP handlers for authentication, client registration, and freelancer registration.

### `schema.sql`

Development database schema for the PostgreSQL database.

## Database Design

Clients and freelancers are represented as separate entities because they have different roles, access patterns, and platform behavior.

Email identity is centralized through the `emails` table to enforce global email uniqueness.

```text
                    emails
                       │
             ┌─────────┴─────────┐
             │                   │
          clients           freelancers
```

Each account stores a reference to its corresponding email record.

Passwords are never stored directly. Passwords are hashed using bcrypt before being persisted.

## Authentication Flow

```text
Client / Freelancer
        │
        │ email + password
        ▼
     POST /login
        │
        ▼
 Find account by email
        │
        ▼
 Compare password with bcrypt hash
        │
        ▼
 Generate signed JWT
        │
        ▼
 Return access token
```

The JWT contains the authenticated account ID and role.

## API Endpoints

### Register Client

```http
POST /clients
```

Example request:

```json
{
  "username": "client1",
  "email": "client@example.com",
  "password": "password123"
}
```

### Register Freelancer

```http
POST /freelancers
```

Example request:

```json
{
  "username": "freelancer1",
  "email": "freelancer@example.com",
  "password": "password123"
}
```

### Login

```http
POST /login
```

Example request:

```json
{
  "email": "client@example.com",
  "password": "password123"
}
```

Successful authentication returns a JWT access token.

## Environment Variables

Create a `.env` file locally:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=freelance_app
DB_PASSWORD=your_password
DB_NAME=freelance_dev

JWT_SECRET=your_jwt_secret
```

`.env` should **never be committed to Git**.

## Running Locally

### 1. Clone the repository

```bash
git clone <repository-url>
cd freelance-backend
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Configure environment variables

Create the `.env` file with your PostgreSQL and JWT configuration.

### 4. Create the development database

Create the PostgreSQL database and run:

```text
db/schema.sql
```

### 5. Start the server

```bash
go run .
```

The API runs on:

```text
http://localhost:8080
```

## Development Database

The included `schema.sql` is intended for **development testing**.

It recreates the account-related tables and therefore should **not be executed against a production database containing real data**.

## Project Roadmap

- [x] PostgreSQL connection
- [x] Client registration
- [x] Freelancer registration
- [x] Password hashing
- [x] Login
- [x] JWT generation
- [ ] Authentication middleware
- [ ] Authorization
- [ ] Job creation
- [ ] Freelancer applications
- [ ] Project lifecycle

## Inspiration

The project is inspired by payment and marketplace workflows observed in platforms such as **Razorpay** and **FixMyitch**. The architecture and implementation are developed independently.

## Project Status

This project is actively being developed as a learning and portfolio project focused on understanding backend architecture, authentication, PostgreSQL, transactions, and eventually escrow workflows.
