# Crypto Exchange

This project is a learning exercise aimed at understanding the inner workings of cryptocurrency exchanges. It serves as a practical study of how exchanges operate, including order matching, trading mechanics, and market dynamics.

## Purpose

The primary goal of this project is to gain a deeper understanding of:
- How cryptocurrency exchanges function
- Order book mechanics and matching algorithms
- Trading pair management
- Market making concepts
- Exchange architecture and design patterns

This is not a production-ready exchange, but rather an educational tool to explore and understand the complexities involved in building and operating a cryptocurrency exchange.

## Project Structure

The project is organized into several key components:
- Order book management
- Trading pair handling
- Order matching engine
- Market data processing

## Getting Started

### Prerequisites
- Go 1.24 or later
- Ganache (local Ethereum blockchain)

### Setup
1. Start Ganache in deterministic mode:
```bash
ganache -d
```

2. Build and run the exchange:
```bash
make run
```

The exchange will start on port 3000.

## API Endpoints

### Get Order Book
```
GET /book/:market
```
Returns the current order book for the specified market (currently only ETH is supported).

Response:
```json
{
    "totalBidVolume": 0,
    "totalAskVolume": 0,
    "asks": [],
    "bids": []
}
```

### Place Order
```
POST /order
```
Place a new order in the exchange.

Request body:
```json
{
    "userID": 7,
    "type": "LIMIT", // or "MARKET"
    "bid": true, // true for buy, false for sell
    "size": 10.0,
    "price": 10000.0, // required for LIMIT orders
    "market": "ETH"
}
```

### Cancel Order
```
DELETE /order/:id
```
Cancel an existing order by its ID.

## Disclaimer

This is an educational project and should not be used for actual trading or production purposes. It lacks many security features and optimizations required for a real-world exchange.
