"""Tests for Pydantic models in bm_photo_client.models."""

from datetime import datetime
from typing import List

import pytest
from pydantic import ValidationError

from bm_photo_client.models import (
    PhotoSummary,
    PhotoDetail,
    Pagination,
    BrowsePhotosResponse,
)


class TestPhotoSummary:
    """Test suite for PhotoSummary model."""

    def test_create_with_all_required_fields(self):
        """Test creating PhotoSummary with all required fields provided."""
        photo = PhotoSummary(
            photo_id="photo-001",
            route_id="NR-001",
            lane_code="L1",
            survey_year=2024,
            gcs_url="gs://bucket/photos/photo-001.jpg",
            uploaded_at=datetime(2024, 1, 15, 10, 30, 0),
            file_name="photo-001.jpg",
        )

        assert photo.photo_id == "photo-001"
        assert photo.route_id == "NR-001"
        assert photo.lane_code == "L1"
        assert photo.survey_year == 2024
        assert photo.gcs_url == "gs://bucket/photos/photo-001.jpg"
        assert photo.uploaded_at == datetime(2024, 1, 15, 10, 30, 0)
        assert photo.file_name == "photo-001.jpg"
        assert photo.sta_value is None

    def test_create_with_optional_sta_value(self):
        """Test creating PhotoSummary with optional sta_value provided."""
        photo = PhotoSummary(
            photo_id="photo-002",
            route_id="NR-002",
            lane_code="L2",
            sta_value=1234.56,
            survey_year=2024,
            gcs_url="gs://bucket/photos/photo-002.jpg",
            uploaded_at=datetime(2024, 2, 20, 14, 45, 0),
            file_name="photo-002.jpg",
        )

        assert photo.photo_id == "photo-002"
        assert photo.sta_value == 1234.56

    def test_datetime_parsing_from_iso_string(self):
        """Test that uploaded_at accepts ISO 8601 formatted datetime strings."""
        iso_datetime = "2024-03-10T08:30:00Z"
        photo = PhotoSummary(
            photo_id="photo-003",
            route_id="NR-003",
            lane_code="L1",
            survey_year=2024,
            gcs_url="gs://bucket/photos/photo-003.jpg",
            uploaded_at=iso_datetime,
            file_name="photo-003.jpg",
        )

        # ISO string with 'Z' suffix creates a timezone-aware datetime
        assert photo.uploaded_at.year == 2024
        assert photo.uploaded_at.month == 3
        assert photo.uploaded_at.day == 10
        assert photo.uploaded_at.hour == 8
        assert photo.uploaded_at.minute == 30
        assert photo.uploaded_at.second == 0

    def test_datetime_parsing_with_milliseconds(self):
        """Test that uploaded_at correctly parses ISO string with milliseconds."""
        iso_datetime = "2024-04-15T12:00:00.123456Z"
        photo = PhotoSummary(
            photo_id="photo-004",
            route_id="NR-001",
            lane_code="L1",
            survey_year=2024,
            gcs_url="gs://bucket/photos/photo-004.jpg",
            uploaded_at=iso_datetime,
            file_name="photo-004.jpg",
        )

        assert photo.uploaded_at.year == 2024
        assert photo.uploaded_at.month == 4
        assert photo.uploaded_at.day == 15
        assert photo.uploaded_at.hour == 12
        assert photo.uploaded_at.minute == 0
        assert photo.uploaded_at.second == 0

    def test_validation_error_for_missing_required_fields(self):
        """Test that ValidationError is raised when required fields are missing."""
        with pytest.raises(ValidationError) as exc_info:
            PhotoSummary(
                route_id="NR-001",
                lane_code="L1",
                survey_year=2024,
                gcs_url="gs://bucket/photos/photo.jpg",
                uploaded_at=datetime.now(),
                file_name="photo.jpg",
            )

        errors = exc_info.value.errors()
        assert any(err["loc"] == ("photo_id",) for err in errors)

    def test_validation_error_for_multiple_missing_fields(self):
        """Test that ValidationError lists all missing required fields."""
        with pytest.raises(ValidationError) as exc_info:
            PhotoSummary(
                route_id="NR-001",
                # missing: photo_id, lane_code, survey_year, gcs_url, uploaded_at, file_name
            )

        errors = exc_info.value.errors()
        error_loc = [err["loc"] for err in errors]
        assert ("photo_id",) in error_loc
        assert ("lane_code",) in error_loc
        assert ("survey_year",) in error_loc
        assert ("gcs_url",) in error_loc
        assert ("uploaded_at",) in error_loc
        assert ("file_name",) in error_loc


class TestPhotoDetail:
    """Test suite for PhotoDetail model."""

    def test_create_with_all_fields(self):
        """Test creating PhotoDetail with all fields including optional ones."""
        photo = PhotoDetail(
            photo_id="detail-001",
            route_id="NR-001",
            lane_code="L1",
            latitude=-6.2088,
            longitude=106.8456,
            sta_value=5678.90,
            sta_source="GPS",
            file_format="jpg",
            file_size_bytes=1024000,
            description="Photo at kilometer 56",
            tags=["route", "survey", "indonesia"],
            uploaded_at=datetime(2024, 5, 1, 9, 0, 0),
            download_url="https://storage.googleapis.com/bucket/photo-001.jpg?token=abc",
        )

        assert photo.photo_id == "detail-001"
        assert photo.route_id == "NR-001"
        assert photo.lane_code == "L1"
        assert photo.latitude == -6.2088
        assert photo.longitude == 106.8456
        assert photo.sta_value == 5678.90
        assert photo.sta_source == "GPS"
        assert photo.file_format == "jpg"
        assert photo.file_size_bytes == 1024000
        assert photo.description == "Photo at kilometer 56"
        assert photo.tags == ["route", "survey", "indonesia"]
        assert photo.uploaded_at == datetime(2024, 5, 1, 9, 0, 0)
        assert "download_url" in photo.model_dump()

    def test_create_with_optional_fields_none(self):
        """Test creating PhotoDetail with optional fields set to None."""
        photo = PhotoDetail(
            photo_id="detail-002",
            route_id="NR-002",
            lane_code="L2",
            latitude=None,
            longitude=None,
            sta_value=None,
            sta_source=None,
            file_format="png",
            file_size_bytes=2048,
            description=None,
            uploaded_at=datetime(2024, 5, 2, 10, 0, 0),
            download_url="https://storage.googleapis.com/bucket/photo-002.jpg",
        )

        assert photo.latitude is None
        assert photo.longitude is None
        assert photo.sta_value is None
        assert photo.sta_source is None
        assert photo.description is None
        assert photo.tags == []  # defaults to empty list

    def test_tags_as_list(self):
        """Test that tags field accepts a list of strings."""
        tags_list: List[str] = ["mountain", "curve", "bridge", "intersection"]

        photo = PhotoDetail(
            photo_id="detail-003",
            route_id="NR-003",
            lane_code="L3",
            file_format="jpg",
            file_size_bytes=512000,
            tags=tags_list,
            uploaded_at=datetime(2024, 5, 3, 11, 0, 0),
            download_url="https://storage.googleapis.com/bucket/photo-003.jpg",
        )

        assert photo.tags == ["mountain", "curve", "bridge", "intersection"]
        assert len(photo.tags) == 4

    def test_tags_default_to_empty_list(self):
        """Test that tags defaults to an empty list when not provided."""
        photo = PhotoDetail(
            photo_id="detail-004",
            route_id="NR-001",
            lane_code="L1",
            file_format="jpg",
            file_size_bytes=256000,
            uploaded_at=datetime(2024, 5, 4, 12, 0, 0),
            download_url="https://storage.googleapis.com/bucket/photo-004.jpg",
        )

        assert photo.tags == []
        assert isinstance(photo.tags, list)

    def test_latitude_valid_range_positive(self):
        """Test latitude validation accepts valid positive value (0 to 90)."""
        photo = PhotoDetail(
            photo_id="detail-005",
            route_id="NR-001",
            lane_code="L1",
            latitude=45.0,
            file_format="jpg",
            file_size_bytes=100,
            uploaded_at=datetime.now(),
            download_url="https://example.com/photo.jpg",
        )

        assert photo.latitude == 45.0

    def test_latitude_valid_range_negative(self):
        """Test latitude validation accepts valid negative value (-90 to 0)."""
        photo = PhotoDetail(
            photo_id="detail-006",
            route_id="NR-001",
            lane_code="L1",
            latitude=-45.0,
            file_format="jpg",
            file_size_bytes=100,
            uploaded_at=datetime.now(),
            download_url="https://example.com/photo.jpg",
        )

        assert photo.latitude == -45.0

    def test_latitude_boundary_value_90(self):
        """Test latitude validation accepts boundary value of 90."""
        photo = PhotoDetail(
            photo_id="detail-007",
            route_id="NR-001",
            lane_code="L1",
            latitude=90.0,
            file_format="jpg",
            file_size_bytes=100,
            uploaded_at=datetime.now(),
            download_url="https://example.com/photo.jpg",
        )

        assert photo.latitude == 90.0

    def test_latitude_boundary_value_minus_90(self):
        """Test latitude validation accepts boundary value of -90."""
        photo = PhotoDetail(
            photo_id="detail-008",
            route_id="NR-001",
            lane_code="L1",
            latitude=-90.0,
            file_format="jpg",
            file_size_bytes=100,
            uploaded_at=datetime.now(),
            download_url="https://example.com/photo.jpg",
        )

        assert photo.latitude == -90.0

    def test_latitude_validation_error_above_90(self):
        """Test latitude validation rejects values above 90."""
        with pytest.raises(ValidationError) as exc_info:
            PhotoDetail(
                photo_id="detail-009",
                route_id="NR-001",
                lane_code="L1",
                latitude=91.0,
                file_format="jpg",
                file_size_bytes=100,
                uploaded_at=datetime.now(),
                download_url="https://example.com/photo.jpg",
            )

        errors = exc_info.value.errors()
        assert any("less than or equal to 90" in err["msg"] for err in errors)

    def test_latitude_validation_error_below_minus_90(self):
        """Test latitude validation rejects values below -90."""
        with pytest.raises(ValidationError) as exc_info:
            PhotoDetail(
                photo_id="detail-010",
                route_id="NR-001",
                lane_code="L1",
                latitude=-91.0,
                file_format="jpg",
                file_size_bytes=100,
                uploaded_at=datetime.now(),
                download_url="https://example.com/photo.jpg",
            )

        errors = exc_info.value.errors()
        assert any("greater than or equal to -90" in err["msg"] for err in errors)

    def test_longitude_valid_range_positive(self):
        """Test longitude validation accepts valid positive value (0 to 180)."""
        photo = PhotoDetail(
            photo_id="detail-011",
            route_id="NR-001",
            lane_code="L1",
            longitude=120.5,
            file_format="jpg",
            file_size_bytes=100,
            uploaded_at=datetime.now(),
            download_url="https://example.com/photo.jpg",
        )

        assert photo.longitude == 120.5

    def test_longitude_valid_range_negative(self):
        """Test longitude validation accepts valid negative value (-180 to 0)."""
        photo = PhotoDetail(
            photo_id="detail-012",
            route_id="NR-001",
            lane_code="L1",
            longitude=-100.0,
            file_format="jpg",
            file_size_bytes=100,
            uploaded_at=datetime.now(),
            download_url="https://example.com/photo.jpg",
        )

        assert photo.longitude == -100.0

    def test_longitude_boundary_value_180(self):
        """Test longitude validation accepts boundary value of 180."""
        photo = PhotoDetail(
            photo_id="detail-013",
            route_id="NR-001",
            lane_code="L1",
            longitude=180.0,
            file_format="jpg",
            file_size_bytes=100,
            uploaded_at=datetime.now(),
            download_url="https://example.com/photo.jpg",
        )

        assert photo.longitude == 180.0

    def test_longitude_boundary_value_minus_180(self):
        """Test longitude validation accepts boundary value of -180."""
        photo = PhotoDetail(
            photo_id="detail-014",
            route_id="NR-001",
            lane_code="L1",
            longitude=-180.0,
            file_format="jpg",
            file_size_bytes=100,
            uploaded_at=datetime.now(),
            download_url="https://example.com/photo.jpg",
        )

        assert photo.longitude == -180.0

    def test_longitude_validation_error_above_180(self):
        """Test longitude validation rejects values above 180."""
        with pytest.raises(ValidationError) as exc_info:
            PhotoDetail(
                photo_id="detail-015",
                route_id="NR-001",
                lane_code="L1",
                longitude=181.0,
                file_format="jpg",
                file_size_bytes=100,
                uploaded_at=datetime.now(),
                download_url="https://example.com/photo.jpg",
            )

        errors = exc_info.value.errors()
        assert any("less than or equal to 180" in err["msg"] for err in errors)

    def test_longitude_validation_error_below_minus_180(self):
        """Test longitude validation rejects values below -180."""
        with pytest.raises(ValidationError) as exc_info:
            PhotoDetail(
                photo_id="detail-016",
                route_id="NR-001",
                lane_code="L1",
                longitude=-181.0,
                file_format="jpg",
                file_size_bytes=100,
                uploaded_at=datetime.now(),
                download_url="https://example.com/photo.jpg",
            )

        errors = exc_info.value.errors()
        assert any("greater than or equal to -180" in err["msg"] for err in errors)

    def test_file_size_bytes_must_be_non_negative(self):
        """Test that file_size_bytes must be >= 0."""
        with pytest.raises(ValidationError) as exc_info:
            PhotoDetail(
                photo_id="detail-017",
                route_id="NR-001",
                lane_code="L1",
                file_format="jpg",
                file_size_bytes=-1,
                uploaded_at=datetime.now(),
                download_url="https://example.com/photo.jpg",
            )

        errors = exc_info.value.errors()
        assert any("greater than or equal to 0" in err["msg"] for err in errors)


class TestPagination:
    """Test suite for Pagination model."""

    def test_create_with_valid_values(self):
        """Test creating Pagination with valid values."""
        pagination = Pagination(
            current_page=1,
            per_page=20,
            total_count=100,
            total_pages=5,
        )

        assert pagination.current_page == 1
        assert pagination.per_page == 20
        assert pagination.total_count == 100
        assert pagination.total_pages == 5

    def test_total_pages_can_be_zero(self):
        """Test that total_pages can be 0 for empty results."""
        pagination = Pagination(
            current_page=1,
            per_page=20,
            total_count=0,
            total_pages=0,
        )

        assert pagination.total_pages == 0
        assert pagination.total_count == 0

    def test_total_pages_greater_than_or_equal_to_zero(self):
        """Test that total_pages must be >= 0 (validation constraint)."""
        pagination = Pagination(
            current_page=1,
            per_page=10,
            total_count=50,
            total_pages=5,
        )

        assert pagination.total_pages >= 0

    def test_current_page_must_be_at_least_1(self):
        """Test that current_page must be >= 1."""
        with pytest.raises(ValidationError) as exc_info:
            Pagination(
                current_page=0,
                per_page=10,
                total_count=50,
                total_pages=5,
            )

        errors = exc_info.value.errors()
        assert any("greater than or equal to 1" in err["msg"] for err in errors)

    def test_per_page_must_be_at_least_1(self):
        """Test that per_page must be >= 1."""
        with pytest.raises(ValidationError) as exc_info:
            Pagination(
                current_page=1,
                per_page=0,
                total_count=50,
                total_pages=5,
            )

        errors = exc_info.value.errors()
        assert any("greater than or equal to 1" in err["msg"] for err in errors)

    def test_total_count_must_be_greater_than_or_equal_to_zero(self):
        """Test that total_count must be >= 0."""
        pagination = Pagination(
            current_page=1,
            per_page=10,
            total_count=0,
            total_pages=0,
        )

        assert pagination.total_count >= 0

    def test_pagination_with_large_page_numbers(self):
        """Test pagination with large but valid page numbers."""
        pagination = Pagination(
            current_page=1000,
            per_page=50,
            total_count=50000,
            total_pages=1000,
        )

        assert pagination.current_page == 1000
        assert pagination.total_pages == 1000


class TestBrowsePhotosResponse:
    """Test suite for BrowsePhotosResponse model."""

    def test_create_with_photos_list_and_pagination(self):
        """Test creating BrowsePhotosResponse with photos list and pagination."""
        photos = [
            PhotoSummary(
                photo_id="photo-001",
                route_id="NR-001",
                lane_code="L1",
                survey_year=2024,
                gcs_url="gs://bucket/photo-001.jpg",
                uploaded_at=datetime(2024, 1, 1, 10, 0, 0),
                file_name="photo-001.jpg",
            ),
            PhotoSummary(
                photo_id="photo-002",
                route_id="NR-001",
                lane_code="L2",
                sta_value=500.0,
                survey_year=2024,
                gcs_url="gs://bucket/photo-002.jpg",
                uploaded_at=datetime(2024, 1, 2, 10, 0, 0),
                file_name="photo-002.jpg",
            ),
        ]

        pagination = Pagination(
            current_page=1,
            per_page=10,
            total_count=2,
            total_pages=1,
        )

        response = BrowsePhotosResponse(
            photos=photos,
            pagination=pagination,
        )

        assert len(response.photos) == 2
        assert response.photos[0].photo_id == "photo-001"
        assert response.photos[1].photo_id == "photo-002"
        assert response.pagination.current_page == 1
        assert response.pagination.total_count == 2

    def test_create_with_empty_photos_list(self):
        """Test creating BrowsePhotosResponse with empty photos list."""
        pagination = Pagination(
            current_page=1,
            per_page=10,
            total_count=0,
            total_pages=0,
        )

        response = BrowsePhotosResponse(
            photos=[],
            pagination=pagination,
        )

        assert response.photos == []
        assert len(response.photos) == 0
        assert response.pagination.total_count == 0
        assert response.pagination.total_pages == 0

    def test_photos_list_populated_by_name(self):
        """Test that photos field can be populated using alias 'photos'."""
        data = {
            "photos": [
                {
                    "photo_id": "photo-003",
                    "route_id": "NR-002",
                    "lane_code": "L1",
                    "survey_year": 2024,
                    "gcs_url": "gs://bucket/photo-003.jpg",
                    "uploaded_at": "2024-02-15T08:00:00Z",
                    "file_name": "photo-003.jpg",
                }
            ],
            "pagination": {
                "current_page": 1,
                "per_page": 20,
                "total_count": 1,
                "total_pages": 1,
            },
        }

        response = BrowsePhotosResponse.model_validate(data)

        assert len(response.photos) == 1
        assert response.photos[0].photo_id == "photo-003"

    def test_nested_pagination_fields(self):
        """Test that pagination nested fields are properly validated."""
        response = BrowsePhotosResponse(
            photos=[],
            pagination=Pagination(
                current_page=5,
                per_page=25,
                total_count=120,
                total_pages=5,
            ),
        )

        assert response.pagination.current_page == 5
        assert response.pagination.per_page == 25
        assert response.pagination.total_count == 120
        assert response.pagination.total_pages == 5
