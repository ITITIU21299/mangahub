# Manga Database Data

This directory contains JSON data files for populating the MangaHub database.

## Files

- `manga.json` - Contains 111 popular manga series with complete metadata

## Data Structure

Each manga entry follows this structure:

```json
{
  "id": "one-piece",
  "title": "One Piece",
  "author": "Eiichiro Oda",
  "genres": ["Action", "Adventure", "Shounen", "Comedy", "Drama"],
  "status": "Ongoing",
  "total_chapters": 1100,
  "description": "Monkey D. Luffy sets off on an adventure...",
  "cover_url": "https://example.com/covers/one-piece.jpg"
}
```

## Genre Distribution

The dataset includes manga across major genres:
- **Shounen**: Action, Adventure, Comedy, Sports, Supernatural
- **Shoujo**: Romance, School, Slice of Life
- **Seinen**: Drama, Psychological, Thriller, Sci-Fi
- **Josei**: Romance, Slice of Life, Drama
- **Sports**: Basketball, Soccer, Tennis, Cycling, Figure Skating
- **Horror**: Supernatural, Psychological, Thriller
- **Sci-Fi**: Mecha, Space, Cyberpunk
- **Slice of Life**: Comedy, Drama, School

## Status Types

- `Ongoing` - Currently being published
- `Completed` - Finished publication
- `Hiatus` - Temporarily paused

## Seeding the Database

To populate the database with this data, run:

```bash
go run ./cmd/seed
```

This will read `data/manga.json` and insert all entries into the SQLite database (`mangahub.db`).

## Adding More Data

To add more manga entries:

1. Edit `data/manga.json` and add new entries following the same structure
2. Run the seed script again: `go run ./cmd/seed`

The seed script uses `INSERT OR REPLACE`, so running it multiple times will update existing entries and add new ones.

