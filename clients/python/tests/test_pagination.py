"""Tests for the auto_paginate function in bm_photo_client._pagination."""

from datetime import datetime
from unittest.mock import MagicMock, call

import pytest

from bm_photo_client._pagination import auto_paginate
from bm_photo_client.models import BrowsePhotosResponse, PhotoSummary, Pagination


def make_photo_summary(photo_id: str, route_id: str = "NR-001") -> PhotoSummary:
    """Helper to create a PhotoSummary for testing."""
    return PhotoSummary(
        photo_id=photo_id,
        route_id=route_id,
        lane_code="L1",
        sta_value=100.0,
        survey_year=2024,
        gcs_url=f"https://storage.googleapis.com/bm-photos/{photo_id}.jpg",
        uploaded_at=datetime(2024, 1, 1, 12, 0, 0),
        file_name=f"{photo_id}.jpg",
    )


def make_response(
    photos: list[PhotoSummary], total_pages: int, current_page: int = 1
) -> BrowsePhotosResponse:
    """Helper to create a BrowsePhotosResponse for testing."""
    return BrowsePhotosResponse(
        photos=photos,
        pagination=Pagination(
            current_page=current_page,
            per_page=10,
            total_count=len(photos),
            total_pages=total_pages,
        ),
    )


class TestAutoPaginate:
    """Tests for the auto_paginate function."""

    def test_single_page_result(self):
        """Test that auto_paginate returns all photos when there is only one page.

        Verifies that:
        - All 5 photos are returned
        - fetch_page is called exactly once with page=1
        """
        mock_fetch_page = MagicMock()

        photos = [make_photo_summary(f"photo-{i}") for i in range(5)]
        mock_fetch_page.return_value = make_response(
            photos, total_pages=1, current_page=1
        )

        result = auto_paginate(mock_fetch_page)

        assert len(result) == 5
        assert result == photos
        mock_fetch_page.assert_called_once_with(page=1)

    def test_multiple_pages(self):
        """Test that auto_paginate correctly iterates through multiple pages.

        Verifies that:
        - All 25 photos across 3 pages are returned
        - fetch_page is called 3 times with correct page numbers (1, 2, 3)
        """
        mock_fetch_page = MagicMock()

        page1_photos = [make_photo_summary(f"photo-{i}") for i in range(10)]
        page2_photos = [make_photo_summary(f"photo-{i}") for i in range(10, 20)]
        page3_photos = [make_photo_summary(f"photo-{i}") for i in range(20, 25)]

        mock_fetch_page.side_effect = [
            make_response(page1_photos, total_pages=3, current_page=1),
            make_response(page2_photos, total_pages=3, current_page=2),
            make_response(page3_photos, total_pages=3, current_page=3),
        ]

        result = auto_paginate(mock_fetch_page)

        assert len(result) == 25
        mock_fetch_page.assert_has_calls(
            [
                call(page=1),
                call(page=2),
                call(page=3),
            ]
        )
        assert mock_fetch_page.call_count == 3

    def test_empty_result(self):
        """Test that auto_paginate returns an empty list when no photos exist.

        Verifies that:
        - An empty list is returned
        - fetch_page is called once
        """
        mock_fetch_page = MagicMock()

        mock_fetch_page.return_value = make_response([], total_pages=1, current_page=1)

        result = auto_paginate(mock_fetch_page)

        assert result == []
        mock_fetch_page.assert_called_once_with(page=1)

    def test_with_additional_kwargs(self):
        """Test that additional kwargs are passed correctly to fetch_page.

        Verifies that:
        - route_id and survey_year kwargs are passed to fetch_page
        - The pagination logic still works correctly with kwargs
        """
        mock_fetch_page = MagicMock()

        photos = [make_photo_summary("photo-1", route_id="NR-001")]
        mock_fetch_page.return_value = make_response(
            photos, total_pages=1, current_page=1
        )

        result = auto_paginate(mock_fetch_page, route_id="NR-001", survey_year=2024)

        assert len(result) == 1
        mock_fetch_page.assert_called_once_with(
            page=1, route_id="NR-001", survey_year=2024
        )

    def test_multiple_pages_with_kwargs(self):
        """Test pagination with kwargs preserved across all page calls.

        Verifies that kwargs are passed to every fetch_page call.
        """
        mock_fetch_page = MagicMock()

        page1_photos = [
            make_photo_summary(f"photo-{i}", route_id="NR-001") for i in range(3)
        ]
        page2_photos = [
            make_photo_summary(f"photo-{i}", route_id="NR-001") for i in range(3, 6)
        ]

        mock_fetch_page.side_effect = [
            make_response(page1_photos, total_pages=2, current_page=1),
            make_response(page2_photos, total_pages=2, current_page=2),
        ]

        result = auto_paginate(mock_fetch_page, route_id="NR-001", survey_year=2024)

        assert len(result) == 6
        assert mock_fetch_page.call_count == 2
        mock_fetch_page.assert_has_calls(
            [
                call(page=1, route_id="NR-001", survey_year=2024),
                call(page=2, route_id="NR-001", survey_year=2024),
            ]
        )
