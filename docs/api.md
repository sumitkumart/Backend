# API Reference

All endpoints are JSON over HTTP. Unless mentioned otherwise, timestamps are UTC ISO-8601 strings. Amounts are precise decimals represented as strings to avoid client rounding.

## `POST /reward`

Records that a user received stock units. The event is idempotent based on `eventId`.

**Request**

```json
{
  "userId": "8b0d5bfd-2ef5-4f82-8f6c-a6b57e5eb7b1",
  "symbol": "RELIANCE",
  "shares": "1.250000",
  "eventId": "a56a8ea6-61e1-4d1e-ab9d-79c32fe64e11",
  "rewardedAt": "2024-05-12T05:32:14Z"
}
```

**Response `201 Created`**

```json
{
  "id": "f0858ab1-98b7-4b1f-a087-4fe9d767fba5",
  "userId": "8b0d5bfd-2ef5-4f82-8f6c-a6b57e5eb7b1",
  "symbol": "RELIANCE",
  "shares": "1.250000",
  "grantedPrice": "2511.6500",
  "brokerageInr": "12.56",
  "taxesInr": "10.90",
  "totalCashOutInr": "3151.51",
  "rewardedAt": "2024-05-12T05:32:14Z",
  "createdAt": "2024-05-12T05:33:01Z",
  "eventKey": "a56a8ea6-61e1-4d1e-ab9d-79c32fe64e11"
}
```

Errors: `400` (validation), `409` (duplicate `eventId`), `500`.

## `GET /today-stocks/{userId}`

Returns every reward grant created today (UTC) for the user.

**Response `200 OK`**

```json
[
  {
    "symbol": "RELIANCE",
    "shares": "1.250000",
    "rewardedAt": "2024-05-12T05:32:14Z"
  },
  {
    "symbol": "INFY",
    "shares": "0.750000",
    "rewardedAt": "2024-05-12T06:05:13Z"
  }
]
```

## `GET /historical-inr/{userId}`

Returns one row per past day (up to yesterday) with the INR valuation that was snapshot by the hourly price job.

```json
[
  { "date": "2024-05-10", "totalInr": "14512.33" },
  { "date": "2024-05-11", "totalInr": "15201.04" }
]
```

## `GET /stats/{userId}`

Summarises rewards granted today and the current INR value of the portfolio.

```json
{
  "totalsToday": [
    { "symbol": "RELIANCE", "shares": "1.250000" },
    { "symbol": "INFY", "shares": "0.750000" }
  ],
  "portfolioInr": "25221.74",
  "priceAsOf": "2024-05-12T09:00:00Z"
}
```

`totalsToday` groups by symbol and only covers the current UTC day. `portfolioInr` is recomputed using the latest cached prices; `priceAsOf` tells you when those prices were fetched (flag staleness if too old).

## `GET /portfolio/{userId}`

Detailed holdings per symbol with mark-to-market values.

```json
[
  {
    "symbol": "RELIANCE",
    "shares": "4.750000",
    "avgAcqPriceInr": "2510.2500",
    "currentPriceInr": "2744.10",
    "currentValueInr": "13034.48",
    "unrealizedPnlInr": "1112.00",
    "priceAsOf": "2024-05-12T09:00:00Z"
  }
]
```

Fields:
- `shares` — total fractional units held.
- `avgAcqPriceInr` — weighted average acquisition price per share.
- `currentValueInr` — shares × current price.
- `unrealizedPnlInr` — difference between current value and cost basis.

All endpoints may return `500` for unexpected errors. Error payloads always use `{ "error": "message" }`.
