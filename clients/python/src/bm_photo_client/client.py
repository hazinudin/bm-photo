"""Bina Marga Survey Photo REST API client."""

import time
from typing import Callable, List, Optional

import requests

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
from ._pagination import auto_paginate


class BMPhotoClient:
    """Client for interacting with the Bina Marga Survey Photo REST API.

    This client provides methods for browsing, retrieving, and downloading
    survey photos of Indonesian national routes.

    Args:
        base_url: The base URL of the API server (e.g., "https://api.example.com").
        api_key: The API key for authentication.
        timeout: Request timeout in seconds (default: 30.0).
        cooldown_threshold: Number of requests before triggering a cooldown (default: 2500).
        cooldown_duration: Cooldown duration in seconds (default: 60).

    Example:
        >>> client = BMPhotoClient(
        ...     base_url="https://api.bm-photo.example.com",
        ...     api_key="your-api-key"
        ... )
        >>> photo_ids = client.get_photo_ids(route_id="NR-001", year=2024)
        >>> print(f"Found {len(photo_ids)} photos")
        >>> photo_detail = client.get_photo(photo_ids[0])
        >>> print(f"First photo: {photo_detail.file_name}")
    """

    def __init__(
        self,
        base_url: str,
        api_key: str,
        *,
        timeout: float = 30.0,
        cooldown_threshold: int = 2500,
        cooldown_duration: float = 60.0,
    ) -> None:
        """Initialize the BMPhotoClient.

        Args:
            base_url: The base URL of the API server.
            api_key: The API key for authentication.
            timeout: Request timeout in seconds (default: 30.0).
            cooldown_threshold: Number of requests before triggering a cooldown (default: 2500).
            cooldown_duration: Cooldown duration in seconds (default: 60).
        """
        self._session = requests.Session()
        self._base_url = base_url.rstrip("/")
        self._timeout = timeout
        self._cooldown_threshold = cooldown_threshold
        self._cooldown_duration = cooldown_duration
        self._request_count = 0
        self._last_cooldown_time = 0.0

        self._session.headers.update(
            {
                "X-API-Key": api_key,
                "Content-Type": "application/json",
            }
        )

    def get_photo_ids(self, route_id: str, year: int, **kwargs) -> List[str]:
        """Return all photo IDs for a given route and survey year.

        This is a convenience method that automatically paginates through
        all results and returns just the photo IDs.

        Args:
            route_id: The national route identifier (e.g., "NR-001").
            year: The survey year to filter by.
            **kwargs: Additional query parameters to pass to browse_photos.

        Returns:
            List of photo_id strings for the specified route and year.

        Example:
            >>> ids = client.get_photo_ids("NR-001", 2024)
            >>> print(f"Found {len(ids)} photos for NR-001 in 2024")
        """
        response = auto_paginate(
            self.browse_photos,
            route_id=route_id,
            survey_year=year,
            **kwargs,
        )
        return [photo.photo_id for photo in response]

    def browse_photos(
        self,
        route_id: str,
        *,
        sta_start: Optional[float] = None,
        sta_end: Optional[float] = None,
        lane_code: Optional[str] = None,
        survey_year: Optional[int] = None,
        page: int = 1,
        per_page: int = 100,
    ) -> BrowsePhotosResponse:
        """Browse photos with optional filtering and pagination.

        Sends a GET request to /api/v1/photos with query parameters
        to filter and paginate the results.

        Args:
            route_id: The national route identifier (e.g., "NR-001").
            sta_start: Optional minimum STA value filter.
            sta_end: Optional maximum STA value filter.
            lane_code: Optional lane code filter (e.g., "L1", "L2").
            survey_year: Optional survey year filter.
            page: Page number (1-indexed, default: 1).
            per_page: Number of results per page (default: 100, max: 1000).

        Returns:
            BrowsePhotosResponse containing a list of PhotoSummary objects
            and pagination metadata.

        Raises:
            ValidationError: If request parameters are invalid.
            AuthenticationError: If API key is invalid.
            RateLimitError: If rate limit is exceeded.
            ServerError: If the server encounters an internal error.
        """
        params: dict = {
            "route_id": route_id,
            "page": page,
            "per_page": per_page,
        }

        if sta_start is not None:
            params["sta_start"] = sta_start
        if sta_end is not None:
            params["sta_end"] = sta_end
        if lane_code is not None:
            params["lane_code"] = lane_code
        if survey_year is not None:
            params["survey_year"] = survey_year

        url = f"{self._base_url}/api/v1/photos"
        response = self._session.get(url, params=params, timeout=self._timeout)

        if response.status_code != 200:
            self._handle_error(response)

        return BrowsePhotosResponse.model_validate(response.json())

    def get_photo(self, photo_id: str) -> PhotoDetail:
        """Retrieve detailed information about a specific photo.

        Sends a GET request to /api/v1/photos/{photo_id} to fetch
        the full details of a photo including metadata and download URL.

        Args:
            photo_id: The unique identifier of the photo.

        Returns:
            PhotoDetail object containing the full photo information.

        Raises:
            NotFoundError: If no photo exists with the given photo_id.
            AuthenticationError: If API key is invalid.
            RateLimitError: If rate limit is exceeded.
            ServerError: If the server encounters an internal error.
        """
        url = f"{self._base_url}/api/v1/photos/{photo_id}"
        response = self._session.get(url, timeout=self._timeout)

        if response.status_code == 404:
            raise NotFoundError(f"Photo not found: {photo_id}")

        if response.status_code != 200:
            self._handle_error(response)

        return PhotoDetail.model_validate(response.json())

    def download_photo_url(self, photo_id: str) -> str:
        """Get the download URL for a specific photo.

        Sends a GET request to /api/v1/photos/{photo_id}/download with
        allow_redirects=False to get the Location header value which
        contains the direct download URL.

        Args:
            photo_id: The unique identifier of the photo.

        Returns:
            The redirect URL from the Location header (the direct download URL).

        Raises:
            NotFoundError: If no photo exists with the given photo_id.
            AuthenticationError: If API key is invalid.
            RateLimitError: If rate limit is exceeded.
            ServerError: If the server encounters an internal error.
        """
        url = f"{self._base_url}/api/v1/photos/{photo_id}/download"
        response = self._session.get(url, timeout=self._timeout, allow_redirects=False)

        if response.status_code == 404:
            raise NotFoundError(f"Photo not found: {photo_id}")

        if response.status_code != 302:
            self._handle_error(response)

        return response.headers.get("Location", "")

    def _check_cooldown(self) -> None:
        """Apply cooldown if request count exceeds threshold.

        Increments the request counter and sleeps for the configured
        cooldown duration when the threshold is reached.
        """
        self._request_count += 1

        if self._request_count >= self._cooldown_threshold:
            self._request_count = 0
            self._last_cooldown_time = time.time()
            time.sleep(self._cooldown_duration)

    def update_photo(
        self,
        photo_id: str,
        *,
        latitude: Optional[float] = None,
        longitude: Optional[float] = None,
        sta_value: Optional[float] = None,
        lane_code: Optional[str] = None,
        description: Optional[str] = None,
        tags: Optional[List[str]] = None,
        survey_year: Optional[int] = None,
    ) -> UpdatePhotoResponse:
        """Update metadata for a specific photo.

        Sends a PATCH request to /api/v1/photos/{photo_id} with only
        the provided fields (partial update semantics).

        Args:
            photo_id: The unique identifier of the photo to update.
            latitude: GPS latitude coordinate (-90 to 90). Must be provided
                together with longitude.
            longitude: GPS longitude coordinate (-180 to 180). Must be provided
                together with latitude.
            sta_value: Station value along the route in meters (>= 0).
                When provided, sta_source is automatically set to "user_provided".
            lane_code: Lane code to set (e.g., "L1", "R2").
            description: New description for the photo.
            tags: New tags list. Sending an empty list clears all tags.
            survey_year: Survey year to set (2000 to current year + 1).

        Returns:
            UpdatePhotoResponse with the updated photo metadata.

        Raises:
            NotFoundError: If no photo exists with the given photo_id.
            ValidationError: If request parameters are invalid.
            ForbiddenError: If API key lacks write scope.
            AuthenticationError: If API key is invalid.
            RateLimitError: If rate limit is exceeded.
            ServerError: If the server encounters an internal error.
        """
        self._check_cooldown()

        body: dict = {}
        if latitude is not None:
            body["latitude"] = latitude
        if longitude is not None:
            body["longitude"] = longitude
        if sta_value is not None:
            body["sta_value"] = sta_value
        if lane_code is not None:
            body["lane_code"] = lane_code
        if description is not None:
            body["description"] = description
        if tags is not None:
            body["tags"] = tags
        if survey_year is not None:
            body["survey_year"] = survey_year

        url = f"{self._base_url}/api/v1/photos/{photo_id}"
        response = self._session.patch(url, json=body, timeout=self._timeout)

        if response.status_code == 404:
            raise NotFoundError(f"Photo not found: {photo_id}")

        if response.status_code == 403:
            raise ForbiddenError(
                "API key lacks write scope to update photos",
                code="INSUFFICIENT_SCOPE",
            )

        if response.status_code != 200:
            self._handle_error(response)

        return UpdatePhotoResponse.model_validate(response.json())

    def batch_update_photos(
        self,
        updates: List[BatchUpdateItem],
        *,
        chunk_size: int = 500,
        on_progress: Optional[Callable[[BatchUpdateResponse], None]] = None,
    ) -> BatchUpdateResponse:
        """Update metadata for multiple photos in batch.

        Automatically deduplicates updates by photo_id (keeping the last occurrence),
        chunks them into batches of chunk_size items, sends each batch as a separate
        HTTP request, and merges the results.

        Args:
            updates: List of BatchUpdateItem objects to update.
            chunk_size: Maximum items per HTTP request (default: 500).
            on_progress: Optional callback called after each chunk completes.
                Receives the BatchUpdateResponse for that chunk.

        Returns:
            BatchUpdateResponse with merged results from all chunks.

        Raises:
            ValidationError: If any chunk request fails validation.
            AuthenticationError: If API key is invalid.
            RateLimitError: If rate limit is exceeded.
            ServerError: If the server encounters an internal error.
        """
        if not updates:
            return BatchUpdateResponse(total=0, succeeded=0, failed=0, results=[])

        # Deduplicate by photo_id, keeping the last occurrence
        seen = {}
        for item in updates:
            seen[item.photo_id] = item
        deduped_updates = list(seen.values())

        all_results: List[BatchUpdateItemResult] = []
        total_succeeded = 0
        total_failed = 0

        for i in range(0, len(deduped_updates), chunk_size):
            chunk = deduped_updates[i : i + chunk_size]

            self._check_cooldown()

            body = {"updates": [item.model_dump(exclude_none=True) for item in chunk]}

            url = f"{self._base_url}/api/v1/photos/batch"
            response = self._session.patch(url, json=body, timeout=self._timeout)

            if response.status_code == 403:
                raise ForbiddenError(
                    "API key lacks write scope to update photos",
                    code="INSUFFICIENT_SCOPE",
                )

            if response.status_code != 200:
                self._handle_error(response)

            chunk_response = BatchUpdateResponse.model_validate(response.json())

            all_results.extend(chunk_response.results)
            total_succeeded += chunk_response.succeeded
            total_failed += chunk_response.failed

            if on_progress is not None:
                on_progress(chunk_response)

        return BatchUpdateResponse(
            total=len(deduped_updates),
            succeeded=total_succeeded,
            failed=total_failed,
            results=all_results,
        )

    def _handle_error(self, response: requests.Response) -> None:
        """Parse and raise appropriate exception for error responses.

        Reads the JSON error body and maps HTTP status code along with
        the error code field to the appropriate exception type.

        Args:
            response: The error response from the API.

        Raises:
            ValidationError: For HTTP 400 responses.
            AuthenticationError: For HTTP 401 responses.
            ForbiddenError: For HTTP 403 responses.
            NotFoundError: For HTTP 404 responses.
            RateLimitError: For HTTP 429 responses.
            ServerError: For HTTP 5xx responses.
        """
        try:
            error_data = response.json()
        except Exception:
            error_data = {}

        error_code = error_data.get("code", "")
        error_msg = error_data.get("error", response.text or "Unknown error")
        details = error_data.get("details")

        if response.status_code == 400 or error_code == "VALIDATION_ERROR":
            raise ValidationError(error_msg, code=error_code, details=details)
        elif response.status_code == 401 or error_code in (
            "MISSING_API_KEY",
            "INVALID_API_KEY",
            "INACTIVE_API_KEY",
            "EXPIRED_API_KEY",
        ):
            raise AuthenticationError(error_msg, code=error_code, details=details)
        elif response.status_code == 403 or error_code in (
            "INSUFFICIENT_SCOPE",
            "FORBIDDEN",
        ):
            raise ForbiddenError(error_msg, code=error_code, details=details)
        elif response.status_code == 404 or error_code in (
            "NOT_FOUND",
            "PHOTO_NOT_FOUND",
        ):
            raise NotFoundError(error_msg, code=error_code, details=details)
        elif response.status_code == 429 or error_code in (
            "RATE_LIMIT_EXCEEDED",
            "QUOTA_EXCEEDED",
        ):
            raise RateLimitError(error_msg, code=error_code, details=details)
        elif response.status_code >= 500 or error_code == "INTERNAL_ERROR":
            raise ServerError(error_msg, code=error_code, details=details)
        else:
            raise BMPhotoError(error_msg, code=error_code, details=details)
