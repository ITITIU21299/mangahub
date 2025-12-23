# MangaDex Integration

## Overview

The Go API now integrates with MangaDex to provide access to the full 80,000+ manga catalog, while using the local SQLite database as a cache.

## How It Works

1. **Local Database First**: The API checks the local database for results
2. **MangaDex Fallback**: If local DB has few results (< 50) or no results for a query, it fetches from MangaDex
3. **Automatic Caching**: Results from MangaDex are automatically cached in the local database for faster future queries

## Configuration

### Enable/Disable MangaDex Integration

By default, MangaDex integration is **enabled**. To disable it:

```bash
# Windows PowerShell
$env:MANGAHUB_USE_MANGADEX="false"
go run ./cmd/all-servers
```

Or set it in your environment permanently.

### When MangaDex is Used

The API will fetch from MangaDex when:
- **No query/filter**: Local DB has < 50 results (for "trending/all" view)
- **With query/filter**: Local DB has 0 results for that specific query

### When Local DB is Used

The API uses local DB when:
- Local DB has sufficient results for the query
- MangaDex integration is disabled
- MangaDex API is unavailable (falls back to local DB)

## Benefits

✅ **Full Catalog Access**: Access to 80,000+ manga from MangaDex  
✅ **Fast Local Queries**: Cached results are served instantly  
✅ **Accurate Totals**: MangaDex provides accurate total counts  
✅ **Automatic Caching**: Frequently accessed manga are cached automatically  
✅ **Fallback Support**: Works even if MangaDex is temporarily unavailable  

## Rate Limiting

The MangaDex client includes rate limiting (200ms delay between requests = 5 req/sec) to respect MangaDex API limits.

## Database Growth

As users browse manga, the local database will automatically grow with cached entries. This improves performance over time.

## Example Flow

1. User searches for "One Piece"
2. API checks local DB → Not found
3. API fetches from MangaDex → Returns results
4. Results are cached in local DB
5. Next search for "One Piece" → Served from local DB (faster!)

## Troubleshooting

### MangaDex Not Working

If MangaDex requests fail:
- Check internet connection
- Verify MangaDex API is accessible: `curl https://api.mangadex.org/manga?limit=1`
- Check server logs for error messages
- API will automatically fallback to local DB

### Too Many MangaDex Requests

If you're making too many requests:
- The rate limiter should prevent this (200ms delay)
- Consider increasing the local DB cache by browsing more manga
- Disable MangaDex for testing: `MANGAHUB_USE_MANGADEX=false`

