"""Pytest fixtures for Bina Marga Photo Service API client tests."""

from datetime import datetime
from typing import List

import pytest
import requests_mock

from bm_photo_client.client import BMPhotoClient


@pytest.fixture
def client():
    """Create a BMPhotoClient instance for testing.

    Returns a client configured with a test base URL and API key.
    The client is ready to be used with requests_mock to simulate responses.
    """
    return BMPhotoClient(
        base_url="https://api.example.com",
        api_key="test-api-key-12345",
    )


@pytest.fixture
def sample_photo_summary():
    """Return a dict with sample PhotoSummary data.

    Returns:
        Dict containing representative PhotoSummary fields matching the
        PhotoSummary model schema.
    """
    return {
        "photo_id": "550e8400-e29b-41d4-a716-446655440000",
        "route_id": "NR-001",
        "lane_code": "L1",
        "sta_value": 5.2,
        "survey_year": 2024,
        "gcs_url": "https://storage.googleapis.com/bucket/photos/2024/NR-001/...",
        "uploaded_at": "2024-01-15T10:30:00Z",
        "file_name": "survey_photo_001.jpg",
    }


@pytest.fixture
def sample_photo_detail():
    """Return a dict with sample PhotoDetail data.

    Returns:
        Dict containing representative PhotoDetail fields matching the
        PhotoDetail model schema.
    """
    return {
        "photo_id": "550e8400-e29b-41d4-a716-446655440000",
        "route_id": "NR-001",
        "lane_code": "L1",
        "latitude": -6.2088,
        "longitude": 106.8456,
        "sta_value": 5.2,
        "sta_source": "GPS",
        "file_format": "jpg",
        "file_size_bytes": 1048576,
        "description": "Survey photo of NR-001, L1 at STA 5.2",
        "tags": ["NR-001", "L1", "2024"],
        "uploaded_at": "2024-01-15T10:30:00Z",
        "download_url": "https://storage.googleapis.com/bucket/photos/2024/NR-001/survey_photo_001.jpg?X-Goog-Signature=...",
    }


@pytest.fixture
def sample_pagination():
    """Return a dict with sample Pagination data.

    Returns:
        Dict containing pagination metadata for a typical list response.
    """
    return {
        "current_page": 1,
        "per_page": 100,
        "total_count": 1,
        "total_pages": 1,
    }


@pytest.fixture
def sample_browse_response(sample_photo_summary, sample_pagination):
    """Return a dict with sample BrowsePhotosResponse data.

    Args:
        sample_photo_summary: Fixture providing sample photo summary data.
        sample_pagination: Fixture providing sample pagination data.

    Returns:
        Dict containing a BrowsePhotosResponse with photos list and
        pagination metadata.
    """
    return {
        "photos": [sample_photo_summary],
        "pagination": sample_pagination,
    }
