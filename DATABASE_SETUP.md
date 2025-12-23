# Database Setup Guide

## Issue: "No results" on Discover Page

If you see "no results" even though the API returns 200 OK, the database is likely empty or the server is using a different database file.

## Solution: Populate the Database

### Step 1: Run the Seed Script

From the `mangahub` directory:

```bash
go run ./cmd/seed
```

This will populate the database with 111 manga entries from `data/manga.json`.

**Expected output:**
```
Successfully inserted 111 manga entries into the database.
```

### Step 2: Verify Database Location

The seed script creates/updates `mangahub.db` in the `mangahub` directory.

**Important**: Make sure your server is using the same database file:

- **If running from `mangahub` directory**: Uses `mangahub.db` ✅
- **If running from `mangahub/cmd/all-servers`**: May use `mangahub.db` in that directory ❌

### Step 3: Use Environment Variable (Recommended)

Set the database path explicitly:

```bash
# Windows PowerShell
$env:MANGAHUB_DB_PATH="D:\StudyFY\UniLab\Uni-Lab\NetCen\mangahub\mangahub.db"

# Then run server
go run ./cmd/all-servers
```

Or create a `.env` file in the `mangahub` directory (for reference, servers read from environment):

```env
MANGAHUB_DB_PATH=mangahub.db
```

### Step 4: Restart the Server

After seeding, restart your server:

```bash
# Stop the server (Ctrl+C)
# Then restart
go run ./cmd/all-servers
```

## Verify Database Has Data

You can check if the database has data:

```bash
# Using sqlite3 (if installed)
sqlite3 mangahub/mangahub.db "SELECT COUNT(*) FROM manga;"
# Should return: 111 (or more)
```

## Troubleshooting

### Multiple Database Files

If you have multiple `mangahub.db` files in different directories:

1. **Find all database files:**
   ```powershell
   Get-ChildItem -Path . -Filter "mangahub.db" -Recurse
   ```

2. **Use the one with the most recent timestamp** (the one you just seeded)

3. **Copy it to where the server expects it:**
   ```powershell
   Copy-Item -Path "mangahub\mangahub.db" -Destination "mangahub\cmd\all-servers\mangahub.db" -Force
   ```

### Database Path Issues

The servers now try to find the database in this order:
1. Environment variable `MANGAHUB_DB_PATH`
2. `../../mangahub.db` (if running from `cmd/all-servers`)
3. `../mangahub.db` (if running from `cmd/api-server`)
4. `mangahub.db` (current directory)

### Still No Results?

1. **Check browser console** - Look for the debug log: `[Discover] API Response:`
2. **Check server logs** - Look for database query errors
3. **Verify authentication** - Make sure you're signed in (token in localStorage)
4. **Test API directly:**
   ```bash
   curl http://25.17.216.66:8080/manga?page=1&limit=20
   # (You'll need to add Authorization header with token)
   ```

## Quick Fix Script

Run this to ensure database is in the right place:

```powershell
# From mangahub directory
go run ./cmd/seed
Copy-Item -Path "mangahub.db" -Destination "cmd\all-servers\mangahub.db" -Force
Copy-Item -Path "mangahub.db" -Destination "cmd\api-server\mangahub.db" -Force
```

Then restart your server.

