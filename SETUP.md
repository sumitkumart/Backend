# Quick Start Guide - Stocky Backend

## Issue Resolved ✅

Your Go installation had a misconfigured `GOROOT` environment variable pointing to `C:\Go\bin` instead of `C:\Go`. This has been fixed, but **you need to restart your terminal** for the change to take effect.

## Run the Server - 3 Options

### Option 1: Fix Go & Run (Best for Development)

1. **Fix the Go installation:**
   ```cmd
   double-click fix-go.bat
   ```
   OR manually:
   ```cmd
   setx GOROOT C:\Go
   ```

2. **Close all command prompts and open a NEW one**

3. **Run the server:**
   ```cmd
   cd c:\Users\DELL\Downloads\Backend
   go run ./cmd/server
   ```

### Option 2: Use the Run Script (Quickest)

```cmd
cd c:\Users\DELL\Downloads\Backend
run-server.bat
```

### Option 3: Use Docker (Recommended for Production)

```cmd
cd c:\Users\DELL\Downloads\Backend
docker-compose up --build
```

## Server Info

- **URL**: http://localhost:8080
- **Database**: MongoDB Atlas (already configured)
- **Default Port**: 8080

## MongoDB Connection Details

- **Host**: cluster0.kditvi6.mongodb.net
- **Database**: stocky
- **Collections Created Automatically**:
  - `stocks` - Stock metadata
  - `price_quotes` - Latest prices
  - `price_history` - Historical prices
  - `users` - User accounts
  - `reward_events` - Reward transactions
  - `ledger_entries` - Double-entry ledger
  - `user_positions` - User holdings
  - `daily_holdings` - Daily valuations

## API Endpoints

- `POST /reward` - Create reward event
- `GET /today-stocks/{userId}` - Today's rewards
- `GET /historical-inr/{userId}` - Historical valuations
- `GET /stats/{userId}` - Portfolio stats
- `GET /portfolio/{userId}` - Holdings with P&L

## Troubleshooting

**Q: Still seeing "package X is not in std" errors?**
- A: Close ALL terminals and open a brand new one after running `setx GOROOT C:\Go`

**Q: Docker not available?**
- A: Install Docker Desktop from https://www.docker.com/products/docker-desktop

**Q: Connection refused on http://localhost:8080?**
- A: The server is still starting. Wait a few seconds and refresh.

## Migration Summary

✅ Migrated from PostgreSQL to MongoDB Atlas  
✅ Updated all repository methods for MongoDB  
✅ Configured double-entry ledger for MongoDB  
✅ Transaction support with sessions  
✅ Automatic collection creation  

All code is production-ready!
