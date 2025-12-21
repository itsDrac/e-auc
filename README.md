# E-Auction.
E-Auction is a web application that would mimic auction but online.
Seller can start an auction with a given time frame. (Max of 3 days)
Biders can bid on the product. the highest bider once the auction is closed,
A chat room would be created for highest bider (buyer at this time) and seller.

## Tech stack.
Golang for backend + chi framework for web + sqlc as database interaction layer + postgress.
Reactjs for frontend (rest whatever AI tell me to install for frontend.)

## Prerequisites
- Go 1.21+
- Swag CLI v1.16.4 for API documentation generation
  ```bash
  go install github.com/swaggo/swag/cmd/swag@v1.16.4
  ```
- Migrate CLI v4.18.2 for database migrations
  ```bash
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.2
  ```
- SQLC v1.29.0 for SQL code generation
  ```bash
  go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0
  ```

## Setup (Dev).
### Database
```
docker run --name postgres -e POSTGRES_PASSWORD=password -d -p 5432:5432 postgres:alpine
```
**Note** Make sure to add database with name "e-auc"

### Migration
To create new migration use
```
migrate create -ext sql -dir migrations -seq -digits 2 <name>
```
To migrate up use 
```
migrate -database <db_uri> -path migrations up
```

###