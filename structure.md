# E-Auction Application Structure

This document provides a visual and detailed explanation of how the e-auction application is structured and organized.

## 📁 Directory Structure

```
e-auc/
├── cmd/                          # Application entry points
│   ├── main.go                   # Main entry point (initializes logger, loads .env)
│   └── server/                   # HTTP server setup
│       ├── server.go             # Server initialization and lifecycle
│       └── routes.go             # API route definitions
│
├── internal/                     # Private application code
│   ├── cache/                    # Caching layer
│   │   └── redis.go              # Redis client implementation
│   │
│   ├── database/                 # Database layer (SQLC generated)
│   │   ├── db.go                 # Database connection logic
│   │   ├── models.go             # Generated database models
│   │   ├── querier.go            # Generated query interface
│   │   ├── users.sql.go          # Generated user queries
│   │   └── products.sql.go       # Generated product queries
│   │
│   ├── dependency/               # Dependency injection container
│   │   └── dependencies.go       # Wires up all dependencies
│   │
│   ├── handlers/                 # HTTP handlers (controllers)
│   │   ├── users.go              # User/Auth endpoints
│   │   ├── products.go           # Product endpoints
│   │   ├── helpers.go            # Response helpers
│   │   └── errors.go             # Error definitions
│   │
│   ├── middleware/               # HTTP middleware
│   │   └── auth-middleware.go    # JWT authentication middleware
│   │
│   ├── model/                    # Request/Response DTOs
│   │   └── *.go                  # Data transfer objects
│   │
│   ├── service/                  # Business logic layer
│   │   ├── services.go           # Service container
│   │   ├── auth.go               # Authentication service
│   │   ├── users.go              # User service
│   │   ├── products.go           # Product service
│   │   └── errors.go             # Service error definitions
│   │
│   └── storage/                  # Object storage layer
│       └── storage.go            # MinIO storage implementation
│
├── pkg/                          # Public/shared packages
│   ├── config/                   # Configuration constants
│   │   └── config.go             # JWT claims, durations, etc.
│   │
│   ├── jwt/                      # JWT utilities
│   │   └── jwt.go                # Token generation and validation
│   │
│   ├── logger/                   # Logging utilities
│   │   └── logger.go             # Structured logging helpers
│   │
│   ├── utils/                    # General utilities
│   │   └── utils.go              # Password hashing, env vars, etc.
│   │
│   └── validator/                # Input validation
│       └── validator.go          # Request validation setup
│
├── migrations/                   # Database migrations
│   ├── 01_users_table.up.sql    # User table creation
│   ├── 01_users_table.down.sql  # User table rollback
│   ├── 02_product_table.up.sql  # Product table creation
│   └── 02_product_table.down.sql # Product table rollback
│
├── queries/                      # SQL queries for SQLC
│   ├── users.sql                 # User-related queries
│   └── products.sql              # Product-related queries
│
├── docs/                         # Swagger documentation
│   ├── docs.go                   # Generated swagger docs
│   ├── swagger.json              # Swagger JSON spec
│   └── swagger.yaml              # Swagger YAML spec
│
├── docker/                       # Docker configurations
│   ├── docker-compose.yml        # Multi-service setup (Postgres, MinIO, Redis)
│   └── redis.conf                # Redis configuration file
│
├── .env                          # Environment variables (not in git)
├── .env.example                  # Example environment variables
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
├── sqlc.yml                      # SQLC configuration
├── makefile                      # Development commands
└── README.md                     # Project documentation
```

## 🏗️ Architecture Overview

The application follows a **layered architecture** with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Layer (API)                      │
│         cmd/server/routes.go + internal/handlers/        │
└───────────────────────┬─────────────────────────────────┘
                        │
                        ↓
┌─────────────────────────────────────────────────────────┐
│                  Business Logic Layer                    │
│                   internal/service/                      │
└───────────────────────┬─────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        ↓               ↓               ↓
┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│  Database   │ │   Storage   │ │    Cache    │
│   (SQLC)    │ │   (MinIO)   │ │   (Redis)   │
└─────────────┘ └─────────────┘ └─────────────┘
```

## 🔄 Request Flow

Here's how a typical authenticated API request flows through the system:

```
1. HTTP Request
   │
   ↓
2. Router (Chi) → Global Middlewares (Logger, Recoverer)
   │
   ↓
3. Auth Middleware (JWT validation)
   │
   ↓
4. Handler (internal/handlers/)
   │  • Parses request body
   │  • Validates input
   │  • Extracts user claims from context
   │
   ↓
5. Service Layer (internal/service/)
   │  • Business logic
   │  • Data validation
   │  • Orchestration
   │
   ↓
6. Data Layer
   │  ├→ Database (SQLC generated queries)
   │  ├→ Storage (MinIO for images)
   │  └→ Cache (Redis for sessions/tokens)
   │
   ↓
7. Response
   │  • Success: JSON response with data
   │  • Error: Standardized error response
   │
   ↓
8. HTTP Response
```

## 🧩 Component Relationships

### Dependency Injection Flow

```
main.go
  │
  ├─→ Load .env file
  ├─→ Configure structured logging (slog)
  │
  ↓
server.New()
  │
  ├─→ dependency.NewDependencies()
  │     │
  │     ├─→ Connect to PostgreSQL (pgx)
  │     ├─→ Initialize SQLC Querier
  │     ├─→ Initialize MinIO Storage
  │     ├─→ Initialize Redis Cache
  │     │
  │     ├─→ service.NewServices()
  │     │     ├─→ AuthService (DB + JWT Manager)
  │     │     ├─→ UserService (DB)
  │     │     └─→ ProductService (DB + Storage)
  │     │
  │     └─→ Initialize Handlers
  │           ├─→ UserHandler (UserService + AuthService)
  │           └─→ ProductHandler (ProductService)
  │
  └─→ Build Routes (Chi Router + Handlers)
```

## 📦 Key Components

### 1. **Entry Point** (`cmd/main.go`)
- Loads environment variables
- Configures structured logging (slog)
- Initializes the HTTP server
- Swagger documentation annotations

### 2. **HTTP Server** (`cmd/server/`)
- **server.go**: Initializes server, manages graceful shutdown
- **routes.go**: Defines API routes and middleware chain

### 3. **Dependency Container** (`internal/dependency/`)
- Single source of truth for all dependencies
- Initializes connections (DB, Cache, Storage)
- Wires up services and handlers
- Ensures proper initialization order

### 4. **Handlers** (`internal/handlers/`)
- Receive HTTP requests
- Parse and validate input
- Call appropriate service methods
- Format and send responses
- Handle errors uniformly

**Example Flow:**
```go
Request → Handler.RegisterUser() 
  → Validate input 
  → AuthService.AddUser() 
  → Return response
```

### 5. **Services** (`internal/service/`)
- Contain business logic
- Orchestrate between data sources
- Enforce business rules
- Return domain errors

**Available Services:**
- **AuthService**: User registration, login, JWT management
- **UserService**: User profile operations
- **ProductService**: Product CRUD, bidding logic, image uploads

### 6. **Database Layer** (`internal/database/`)
- **SQLC Generated**: Type-safe SQL queries
- **db.go**: Custom connection pooling and transaction helpers
- **Querier Interface**: Allows for easy mocking in tests

**Query Definitions:**
- `queries/users.sql` → generates `users.sql.go`
- `queries/products.sql` → generates `products.sql.go`

### 7. **Storage Layer** (`internal/storage/`)
- **MinIO Integration**: Object storage for product images
- **Interface-based**: Easy to swap implementations
- Handles bucket creation, file uploads, URL generation

### 8. **Cache Layer** (`internal/cache/`)
- **Redis Integration**: Session management, token blacklisting
- **Connection pooling**: Optimized for performance
- Supports Get/Set/Delete/Ping operations

### 9. **Middleware** (`internal/middleware/`)
- **Auth Middleware**: JWT validation, user context injection
- Extracting Bearer tokens
- Token blacklist checking
- Setting user claims in request context

### 10. **Shared Packages** (`pkg/`)
- **jwt**: Token generation and validation
- **config**: Constants (token durations, context keys)
- **utils**: Password hashing, environment variable helpers
- **validator**: Request validation setup
- **logger**: Structured logging utilities

## 🔐 Authentication Flow

```
Registration:
  POST /api/v1/auth/register
    → UserHandler.RegisterUser()
    → AuthService.AddUser()
    → Hash password
    → Store in database
    → Return user ID

Login:
  POST /api/v1/auth/login
    → UserHandler.LoginUser()
    → AuthService.ValidateUser()
    → Compare password hash
    → Generate JWT token pair
    → Return tokens

Protected Request:
  GET /api/v1/users/profile
    → AuthMiddleware validates JWT
    → Extract user claims
    → Add to request context
    → UserHandler.GetProfile()
    → Service layer operations
    → Return user data

Token Refresh:
  POST /api/v1/auth/refresh
    → Validate refresh token
    → Generate new token pair
    → Return new tokens

Logout:
  POST /api/v1/auth/logout
    → Extract access token
    → Add to Redis blacklist
    → Clear refresh token cookie
```