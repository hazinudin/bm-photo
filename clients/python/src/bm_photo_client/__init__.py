"""
bm_photo_client - Python client library for Bina Marga Survey Photo Service.

A Python client for interacting with the BM Photo API, providing functionality
for managing survey photographs of Indonesian national routes.

Example usage:
    from bm_photo_client import BMPhotoClient

    client = BMPhotoClient(api_key="your-api-key")
    photos = client.browse_photos(route_id="NR-001")
"""

__version__ = "0.1.0"

from .client import BMPhotoClient
from .models import (
    BatchUpdateItem,
    BatchUpdateItemResult,
    BatchUpdateResponse,
    BrowsePhotosResponse,
    PhotoDetail,
    PhotoSummary,
    UpdatePhotoResponse,
)
from .exceptions import (
    BMPhotoError,
    AuthenticationError,
    ForbiddenError,
    NotFoundError,
    ValidationError,
    RateLimitError,
    ServerError,
)

__all__ = [
    # Version
    "__version__",
    # Client
    "BMPhotoClient",
    # Models
    "BatchUpdateItem",
    "BatchUpdateItemResult",
    "BatchUpdateResponse",
    "BrowsePhotosResponse",
    "PhotoDetail",
    "PhotoSummary",
    "UpdatePhotoResponse",
    # Exceptions
    "BMPhotoError",
    "AuthenticationError",
    "ForbiddenError",
    "NotFoundError",
    "ValidationError",
    "RateLimitError",
    "ServerError",
]
