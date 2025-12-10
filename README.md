# E-Auction.
E-Auction is a web application that would mimic auction but online.
Seller can start an auction with a given time frame. (Max of 3 days)
Biders can bid on the product. the highest bider once the auction is closed,
A chat room would be created for highest bider (buyer at this time) and seller.

## Tech stack.
Golang for backend + chi framework for web + sqlc as database interaction layer + postgress.
Reactjs for frontend (rest whatever AI tell me to install for frontend.)

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