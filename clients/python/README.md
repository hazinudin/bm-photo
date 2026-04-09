# Bina Marga Photo Client

Python client for the Bina Marga Survey Photo Service REST API.

## Installation

```bash
uv add git+https://github.com/bina-marga/bm-photo.git#subdirectory=clients/python
```

Or with pip:

```bash
pip install git+https://github.com/bina-marga/bm-photo.git#subdirectory=clients/python
```

## Quick Start

```python
from bm_photo_client import BMPhotoClient

# Create client
client = BMPhotoClient(
    base_url="https://api.example.com",
    api_key="your-api-key"
)

# Get all photo IDs for a route and year
photo_ids = client.get_photo_ids(route_id="NR-001", year=2024)
print(f"Found {len(photo_ids)} photos")

# Get photo details
photo = client.get_photo(photo_id="550e8400-e29b-41d4-a716-446655440000")
print(f"Photo uploaded at: {photo.uploaded_at}")

# Get download URL
download_url = client.download_photo_url(photo_id="550e8400-e29b-41d4-a716-446655440000")
```

## Features

- **Core use case**: `get_photo_ids(route_id, year)` - auto-paginates to return all photo IDs
- Browse photos with filters (route, year, lane, STA range)
- Get individual photo details
- Get signed download URLs
- Proper error handling with specific exception types
- Type-safe Pydantic models

## API Coverage

- `GET /api/v1/photos` - Browse/filter photos
- `GET /api/v1/photos/{photo_id}` - Get photo details
- `GET /api/v1/photos/{photo_id}/download` - Get download URL

## Error Handling

The client raises specific exceptions based on API error codes:

- `AuthenticationError` (401) - Invalid or expired API key
- `ForbiddenError` (403) - Insufficient permissions
- `NotFoundError` (404) - Photo not found
- `ValidationError` (400) - Invalid request parameters
- `RateLimitError` (429) - Rate limit exceeded
- `ServerError` (5xx) - Server-side errors

## Development

```bash
cd clients/python
uv sync --extra dev
uv run pytest
```

## License

MIT
