from typing import Callable, List, TypeVar

from .models import BrowsePhotosResponse, PhotoSummary

T = TypeVar("T")


def auto_paginate(
    fetch_page: Callable[..., BrowsePhotosResponse], **kwargs
) -> List[PhotoSummary]:
    """
    Fetch all pages by iterating through pagination.

    Args:
        fetch_page: A callable that accepts page and **kwargs and returns BrowsePhotosResponse
        **kwargs: Additional arguments to pass to fetch_page (excluding 'page')

    Returns:
        List of all PhotoSummary items across all pages
    """
    all_photos: List[PhotoSummary] = []
    page = 1

    while True:
        response = fetch_page(page=page, **kwargs)

        if response.photos:
            all_photos.extend(response.photos)

        if page >= response.pagination.total_pages or not response.photos:
            break

        page += 1

    return all_photos
