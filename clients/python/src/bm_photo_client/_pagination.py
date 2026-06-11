import time
from typing import Callable, List, TypeVar

from .models import BrowsePhotosResponse, PhotoSummary
from .exceptions import RateLimitError

T = TypeVar("T")


def auto_paginate(
    fetch_page: Callable[..., BrowsePhotosResponse],
    *,
    max_retries: int = 5,
    base_delay: float = 1.0,
    max_delay: float = 60.0,
    **kwargs,
) -> List[PhotoSummary]:
    """
    Fetch all pages by iterating through pagination with automatic retry on rate limits.

    Args:
        fetch_page: A callable that accepts page and **kwargs and returns BrowsePhotosResponse
        max_retries: Maximum number of retry attempts per page when rate limited (default: 5)
        base_delay: Initial delay in seconds before first retry (default: 1.0)
        max_delay: Maximum delay in seconds between retries (default: 60.0)
        **kwargs: Additional arguments to pass to fetch_page (excluding 'page')

    Returns:
        List of all PhotoSummary items across all pages

    Raises:
        RateLimitError: If rate limit is exceeded after all retry attempts
    """
    all_photos: List[PhotoSummary] = []
    page = 1

    while True:
        response = _fetch_with_retry(
            fetch_page,
            page=page,
            max_retries=max_retries,
            base_delay=base_delay,
            max_delay=max_delay,
            **kwargs,
        )

        if response.photos:
            all_photos.extend(response.photos)

        if page >= response.pagination.total_pages or not response.photos:
            break

        page += 1

    return all_photos


def _fetch_with_retry(
    fetch_page: Callable[..., BrowsePhotosResponse],
    page: int,
    *,
    max_retries: int,
    base_delay: float,
    max_delay: float,
    **kwargs,
) -> BrowsePhotosResponse:
    """
    Fetch a single page with exponential backoff retry on rate limits.

    Args:
        fetch_page: A callable that fetches a single page
        page: Page number to fetch
        max_retries: Maximum number of retry attempts
        base_delay: Initial delay in seconds
        max_delay: Maximum delay in seconds
        **kwargs: Additional arguments to pass to fetch_page

    Returns:
        BrowsePhotosResponse for the requested page

    Raises:
        RateLimitError: If rate limit is exceeded after all retries
    """
    last_exception = None
    delay = base_delay

    for attempt in range(max_retries + 1):
        try:
            return fetch_page(page=page, **kwargs)
        except RateLimitError as e:
            last_exception = e
            if attempt == max_retries:
                raise

            time.sleep(delay)
            delay = min(delay * 2, max_delay)
