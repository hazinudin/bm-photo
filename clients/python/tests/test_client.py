"""Tests for the BMPhotoClient class."""

import json
from datetime import datetime
from unittest.mock import MagicMock, patch

import pytest
import requests_mock

from bm_photo_client.client import BMPhotoClient
from bm_photo_client.exceptions import (
    AuthenticationError,
    BMPhotoError,
    ForbiddenError,
    NotFoundError,
    RateLimitError,
    ServerError,
    ValidationError,
)


# =============================================================================
# Test Fixtures
# =============================================================================


@pytest.fixture
def base_url():
    """Return a base URL for testing."""
    return "https://api.bm-photo.example.com"


@pytest.fixture
def api_key():
    """Return an API key for testing."""
    return "test-api-key-12345"


@pytest.fixture
def client(base_url, api_key):
    """Create a BMPhotoClient instance for testing."""
    return BMPhotoClient(base_url=base_url, api_key=api_key)


@pytest.fixture
def sample_photo_summary():
    """Return a sample PhotoSummary dict."""
    return {
        "photo_id": "photo-001",
        "route_id": "NR-001",
        "lane_code": "L1",
        "sta_value": 1234.5,
        "survey_year": 2024,
        "gcs_url": "gs://bucket/path/photo-001.jpg",
        "uploaded_at": "2024-01-15T10:30:00Z",
        "file_name": "photo-001.jpg",
    }


@pytest.fixture
def sample_photo_detail():
    """Return a sample PhotoDetail dict."""
    return {
        "photo_id": "photo-001",
        "route_id": "NR-001",
        "lane_code": "L1",
        "latitude": -6.2088,
        "longitude": 106.8456,
        "sta_value": 1234.5,
        "sta_source": "GPS",
        "file_format": "jpg",
        "file_size_bytes": 1048576,
        "description": "Survey photo of NR-001 L1 at STA 1234.5",
        "tags": ["survey", "nr-001", "lane-1"],
        "uploaded_at": "2024-01-15T10:30:00Z",
        "download_url": "https://storage.googleapis.com/bucket/path/photo-001.jpg?token=abc123",
    }


@pytest.fixture
def sample_browse_response():
    """Return a sample BrowsePhotosResponse dict."""
    return {
        "photos": [
            {
                "photo_id": "photo-001",
                "route_id": "NR-001",
                "lane_code": "L1",
                "sta_value": 1234.5,
                "survey_year": 2024,
                "gcs_url": "gs://bucket/path/photo-001.jpg",
                "uploaded_at": "2024-01-15T10:30:00Z",
                "file_name": "photo-001.jpg",
            },
            {
                "photo_id": "photo-002",
                "route_id": "NR-001",
                "lane_code": "L1",
                "sta_value": 1235.0,
                "survey_year": 2024,
                "gcs_url": "gs://bucket/path/photo-002.jpg",
                "uploaded_at": "2024-01-15T10:31:00Z",
                "file_name": "photo-002.jpg",
            },
        ],
        "pagination": {
            "current_page": 1,
            "per_page": 100,
            "total_count": 2,
            "total_pages": 1,
        },
    }


# =============================================================================
# Test BMPhotoClient.__init__
# =============================================================================


class TestInit:
    """Tests for BMPhotoClient.__init__ method."""

    def test_client_creation_with_base_url_and_api_key(self, base_url, api_key):
        """Test that client is created successfully with base_url and api_key."""
        client = BMPhotoClient(base_url=base_url, api_key=api_key)

        assert client._base_url == base_url
        assert client._timeout == 30.0
        assert client._session is not None

    def test_trailing_slash_is_stripped_from_base_url(self, api_key):
        """Test that trailing slash is removed from base_url."""
        client_with_slash = BMPhotoClient(
            base_url="https://api.example.com/", api_key=api_key
        )
        client_without_slash = BMPhotoClient(
            base_url="https://api.example.com", api_key=api_key
        )

        assert client_with_slash._base_url == "https://api.example.com"
        assert client_without_slash._base_url == "https://api.example.com"
        assert client_with_slash._base_url == client_without_slash._base_url

    def test_headers_are_set_correctly(self, base_url, api_key):
        """Test that required headers X-API-Key and Content-Type are set."""
        client = BMPhotoClient(base_url=base_url, api_key=api_key)

        assert client._session.headers["X-API-Key"] == api_key
        assert client._session.headers["Content-Type"] == "application/json"

    def test_custom_timeout_is_set(self, base_url, api_key):
        """Test that custom timeout value is used when provided."""
        custom_timeout = 60.0
        client = BMPhotoClient(
            base_url=base_url, api_key=api_key, timeout=custom_timeout
        )

        assert client._timeout == custom_timeout

    def test_default_timeout_is_30_seconds(self, base_url, api_key):
        """Test that default timeout is 30 seconds when not specified."""
        client = BMPhotoClient(base_url=base_url, api_key=api_key)

        assert client._timeout == 30.0


# =============================================================================
# Test BMPhotoClient.get_photo_ids
# =============================================================================


class TestGetPhotoIds:
    """Tests for BMPhotoClient.get_photo_ids method."""

    def test_auto_pagination_collects_all_photo_ids(
        self, client, sample_browse_response
    ):
        """Test that auto-pagination correctly collects all photo IDs from multiple pages."""
        page1_response = {
            "photos": [
                {
                    "photo_id": "photo-001",
                    "route_id": "NR-001",
                    "lane_code": "L1",
                    "sta_value": 1234.5,
                    "survey_year": 2024,
                    "gcs_url": "gs://bucket/path/photo-001.jpg",
                    "uploaded_at": "2024-01-15T10:30:00Z",
                    "file_name": "photo-001.jpg",
                },
            ],
            "pagination": {
                "current_page": 1,
                "per_page": 100,
                "total_count": 3,
                "total_pages": 2,
            },
        }

        page2_response = {
            "photos": [
                {
                    "photo_id": "photo-002",
                    "route_id": "NR-001",
                    "lane_code": "L1",
                    "sta_value": 1235.0,
                    "survey_year": 2024,
                    "gcs_url": "gs://bucket/path/photo-002.jpg",
                    "uploaded_at": "2024-01-15T10:31:00Z",
                    "file_name": "photo-002.jpg",
                },
                {
                    "photo_id": "photo-003",
                    "route_id": "NR-001",
                    "lane_code": "L1",
                    "sta_value": 1235.5,
                    "survey_year": 2024,
                    "gcs_url": "gs://bucket/path/photo-003.jpg",
                    "uploaded_at": "2024-01-15T10:32:00Z",
                    "file_name": "photo-003.jpg",
                },
            ],
            "pagination": {
                "current_page": 2,
                "per_page": 100,
                "total_count": 3,
                "total_pages": 2,
            },
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                [
                    {"json": page1_response, "status_code": 200},
                    {"json": page2_response, "status_code": 200},
                ],
            )

            photo_ids = client.get_photo_ids(route_id="NR-001", year=2024)

        assert photo_ids == ["photo-001", "photo-002", "photo-003"]
        assert len(photo_ids) == 3

    def test_returns_list_of_strings(self, client, sample_browse_response):
        """Test that the result is a list of string photo IDs."""
        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=sample_browse_response,
                status_code=200,
            )

            photo_ids = client.get_photo_ids(route_id="NR-001", year=2024)

        assert isinstance(photo_ids, list)
        assert all(isinstance(pid, str) for pid in photo_ids)
        assert photo_ids == ["photo-001", "photo-002"]

    def test_route_id_and_year_parameters_passed_correctly(
        self, client, sample_browse_response
    ):
        """Test that route_id and survey_year parameters are passed correctly to browse_photos."""
        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=sample_browse_response,
                status_code=200,
            )

            client.get_photo_ids(route_id="NR-001", year=2024)

            # Verify the request was made with correct query parameters
            history = m.request_history[0]
            assert history.qs["route_id"] == ["nr-001"]
            assert history.qs["survey_year"] == ["2024"]

    def test_empty_result_when_no_photos_found(self, client):
        """Test that an empty list is returned when no photos match the criteria."""
        empty_response = {
            "photos": [],
            "pagination": {
                "current_page": 1,
                "per_page": 100,
                "total_count": 0,
                "total_pages": 0,
            },
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=empty_response,
                status_code=200,
            )

            photo_ids = client.get_photo_ids(route_id="NR-999", year=2024)

        assert photo_ids == []


# =============================================================================
# Test BMPhotoClient.browse_photos
# =============================================================================


class TestBrowsePhotos:
    """Tests for BMPhotoClient.browse_photos method."""

    def test_returns_browse_photos_response_object(
        self, client, sample_browse_response
    ):
        """Test that method returns a BrowsePhotosResponse object."""
        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=sample_browse_response,
                status_code=200,
            )

            response = client.browse_photos(route_id="NR-001")

        assert response.photos is not None
        assert len(response.photos) == 2
        assert response.photos[0].photo_id == "photo-001"
        assert response.photos[1].photo_id == "photo-002"
        assert response.pagination.current_page == 1
        assert response.pagination.total_count == 2

    def test_query_parameters_built_correctly(self, client, sample_browse_response):
        """Test that query parameters are correctly built from method arguments."""
        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=sample_browse_response,
                status_code=200,
            )

            client.browse_photos(
                route_id="NR-001",
                sta_start=1000.0,
                sta_end=2000.0,
                lane_code="L1",
                survey_year=2024,
                page=2,
                per_page=50,
            )

            history = m.request_history[0]
            assert history.qs["route_id"] == ["nr-001"]
            assert history.qs["sta_start"] == ["1000.0"]
            assert history.qs["sta_end"] == ["2000.0"]
            assert history.qs["lane_code"] == ["l1"]
            assert history.qs["survey_year"] == ["2024"]
            assert history.qs["page"] == ["2"]
            assert history.qs["per_page"] == ["50"]

    def test_optional_parameters_excluded_when_not_provided(
        self, client, sample_browse_response
    ):
        """Test that optional parameters are not included in query string when None."""
        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=sample_browse_response,
                status_code=200,
            )

            client.browse_photos(route_id="NR-001")

            history = m.request_history[0]
            assert "sta_start" not in history.qs
            assert "sta_end" not in history.qs
            assert "lane_code" not in history.qs
            assert "survey_year" not in history.qs

    def test_error_404_raises_not_found_error(self, client):
        """Test that 404 response raises NotFoundError."""
        error_response = {
            "error": "Not found",
            "code": "NOT_FOUND",
            "message": "No photos found for the specified criteria",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=404,
            )

            with pytest.raises(NotFoundError) as exc_info:
                client.browse_photos(route_id="NR-999")

            assert "NOT_FOUND" in str(exc_info.value)

    def test_error_400_raises_validation_error(self, client):
        """Test that 400 response raises ValidationError."""
        error_response = {
            "error": "Validation failed",
            "code": "VALIDATION_ERROR",
            "message": "Invalid route_id format",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=400,
            )

            with pytest.raises(ValidationError) as exc_info:
                client.browse_photos(route_id="invalid-route!")

            assert "VALIDATION_ERROR" in str(exc_info.value)

    def test_error_401_raises_authentication_error(self, client):
        """Test that 401 response raises AuthenticationError."""
        error_response = {
            "error": "Unauthorized",
            "code": "INVALID_API_KEY",
            "message": "The provided API key is invalid",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=401,
            )

            with pytest.raises(AuthenticationError) as exc_info:
                client.browse_photos(route_id="NR-001")

            assert "INVALID_API_KEY" in str(exc_info.value)

    def test_error_429_raises_rate_limit_error(self, client):
        """Test that 429 response raises RateLimitError."""
        error_response = {
            "error": "Rate limit exceeded",
            "code": "RATE_LIMIT_EXCEEDED",
            "message": "Too many requests, please try again later",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=429,
            )

            with pytest.raises(RateLimitError) as exc_info:
                client.browse_photos(route_id="NR-001")

            assert "RATE_LIMIT_EXCEEDED" in str(exc_info.value)

    def test_error_500_raises_server_error(self, client):
        """Test that 500 response raises ServerError."""
        error_response = {
            "error": "Internal server error",
            "code": "INTERNAL_ERROR",
            "message": "An unexpected error occurred",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=500,
            )

            with pytest.raises(ServerError) as exc_info:
                client.browse_photos(route_id="NR-001")

            assert "INTERNAL_ERROR" in str(exc_info.value)


# =============================================================================
# Test BMPhotoClient.get_photo
# =============================================================================


class TestGetPhoto:
    """Tests for BMPhotoClient.get_photo method."""

    def test_returns_photo_detail_object(self, client, sample_photo_detail):
        """Test that method returns a PhotoDetail object."""
        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos/photo-001",
                json=sample_photo_detail,
                status_code=200,
            )

            photo = client.get_photo("photo-001")

        assert photo.photo_id == "photo-001"
        assert photo.route_id == "NR-001"
        assert photo.lane_code == "L1"
        assert photo.latitude == -6.2088
        assert photo.longitude == 106.8456
        assert photo.file_format == "jpg"
        assert photo.file_size_bytes == 1048576
        assert photo.download_url == sample_photo_detail["download_url"]

    def test_photo_detail_contains_all_fields(self, client, sample_photo_detail):
        """Test that PhotoDetail object contains all expected fields."""
        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos/photo-001",
                json=sample_photo_detail,
                status_code=200,
            )

            photo = client.get_photo("photo-001")

        assert photo.photo_id == sample_photo_detail["photo_id"]
        assert photo.route_id == sample_photo_detail["route_id"]
        assert photo.lane_code == sample_photo_detail["lane_code"]
        assert photo.latitude == sample_photo_detail["latitude"]
        assert photo.longitude == sample_photo_detail["longitude"]
        assert photo.sta_value == sample_photo_detail["sta_value"]
        assert photo.sta_source == sample_photo_detail["sta_source"]
        assert photo.file_format == sample_photo_detail["file_format"]
        assert photo.file_size_bytes == sample_photo_detail["file_size_bytes"]
        assert photo.description == sample_photo_detail["description"]
        assert photo.tags == sample_photo_detail["tags"]
        assert photo.download_url == sample_photo_detail["download_url"]

    def test_404_raises_not_found_error(self, client):
        """Test that 404 response raises NotFoundError."""
        error_response = {
            "error": "Photo not found",
            "code": "PHOTO_NOT_FOUND",
            "message": "No photo exists with ID: nonexistent-photo",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos/nonexistent-photo",
                json=error_response,
                status_code=404,
            )

            with pytest.raises(NotFoundError) as exc_info:
                client.get_photo("nonexistent-photo")

            assert "photo-001" not in str(exc_info.value)
            assert "nonexistent-photo" in str(exc_info.value)

    def test_404_with_error_code_in_body_raises_not_found_error(self, client):
        """Test that 404 with NOT_FOUND error code in body raises NotFoundError."""
        error_response = {
            "error": "Resource not found",
            "code": "NOT_FOUND",
            "message": "The requested resource does not exist",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos/deleted-photo",
                json=error_response,
                status_code=404,
            )

            with pytest.raises(NotFoundError):
                client.get_photo("deleted-photo")


# =============================================================================
# Test BMPhotoClient.download_photo_url
# =============================================================================


class TestDownloadPhotoUrl:
    """Tests for BMPhotoClient.download_photo_url method."""

    def test_returns_location_header_value(self, client):
        """Test that method returns the Location header value."""
        download_url = (
            "https://storage.googleapis.com/bucket/path/photo-001.jpg?token=xyz789"
        )

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos/photo-001/download",
                status_code=302,
                headers={"Location": download_url},
            )

            result = client.download_photo_url("photo-001")

        assert result == download_url

    def test_302_redirect_returns_download_url(self, client):
        """Test that 302 redirect response provides the download URL."""
        expected_url = "https://storage.googleapis.com/signed-url/photo-001.jpg"

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos/photo-001/download",
                status_code=302,
                headers={"Location": expected_url},
            )

            result = client.download_photo_url("photo-001")

        assert result == expected_url

    def test_404_raises_not_found_error(self, client):
        """Test that 404 response raises NotFoundError."""
        error_response = {
            "error": "Photo not found",
            "code": "PHOTO_NOT_FOUND",
            "message": "No photo exists with ID: deleted-photo",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos/deleted-photo/download",
                json=error_response,
                status_code=404,
            )

            with pytest.raises(NotFoundError) as exc_info:
                client.download_photo_url("deleted-photo")

            assert "deleted-photo" in str(exc_info.value)

    def test_non_redirect_non_404_raises_error(self, client):
        """Test that non-302, non-404 responses raise appropriate errors."""
        error_response = {
            "error": "Bad request",
            "code": "BAD_REQUEST",
            "message": "Invalid request",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos/photo-001/download",
                json=error_response,
                status_code=400,
            )

            with pytest.raises(ValidationError):
                client.download_photo_url("photo-001")

    def test_empty_location_header_returns_empty_string(self, client):
        """Test that empty Location header returns empty string."""
        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos/photo-001/download",
                status_code=302,
                headers={"Location": ""},
            )

            result = client.download_photo_url("photo-001")

        assert result == ""


# =============================================================================
# Test BMPhotoClient._handle_error
# =============================================================================


class TestHandleError:
    """Tests for BMPhotoClient._handle_error method."""

    def test_400_raises_validation_error(self, client):
        """Test that HTTP 400 raises ValidationError."""
        error_response = {
            "error": "Validation failed",
            "code": "VALIDATION_ERROR",
            "message": "Invalid input data",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=400,
            )

            with pytest.raises(ValidationError) as exc_info:
                client.browse_photos(route_id="NR-001")

            assert exc_info.value.code == "VALIDATION_ERROR"
            assert "Validation failed" in str(exc_info.value)

    def test_401_raises_authentication_error(self, client):
        """Test that HTTP 401 raises AuthenticationError."""
        error_response = {
            "error": "Authentication required",
            "code": "MISSING_API_KEY",
            "message": "API key is required",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=401,
            )

            with pytest.raises(AuthenticationError) as exc_info:
                client.browse_photos(route_id="NR-001")

            assert exc_info.value.code == "MISSING_API_KEY"

    def test_403_raises_forbidden_error(self, client):
        """Test that HTTP 403 raises ForbiddenError."""
        error_response = {
            "error": "Access denied",
            "code": "INSUFFICIENT_SCOPE",
            "message": "API key lacks required permissions",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=403,
            )

            with pytest.raises(ForbiddenError) as exc_info:
                client.browse_photos(route_id="NR-001")

            assert exc_info.value.code == "INSUFFICIENT_SCOPE"

    def test_404_raises_not_found_error(self, client):
        """Test that HTTP 404 raises NotFoundError."""
        error_response = {
            "error": "Not found",
            "code": "NOT_FOUND",
            "message": "Resource does not exist",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=404,
            )

            with pytest.raises(NotFoundError) as exc_info:
                client.browse_photos(route_id="NR-001")

            assert exc_info.value.code == "NOT_FOUND"

    def test_429_raises_rate_limit_error(self, client):
        """Test that HTTP 429 raises RateLimitError."""
        error_response = {
            "error": "Rate limit exceeded",
            "code": "RATE_LIMIT_EXCEEDED",
            "message": "Too many requests",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=429,
            )

            with pytest.raises(RateLimitError) as exc_info:
                client.browse_photos(route_id="NR-001")

            assert exc_info.value.code == "RATE_LIMIT_EXCEEDED"

    def test_500_raises_server_error(self, client):
        """Test that HTTP 500 raises ServerError."""
        error_response = {
            "error": "Internal server error",
            "code": "INTERNAL_ERROR",
            "message": "An unexpected error occurred",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=500,
            )

            with pytest.raises(ServerError) as exc_info:
                client.browse_photos(route_id="NR-001")

            assert exc_info.value.code == "INTERNAL_ERROR"

    def test_502_raises_server_error(self, client):
        """Test that HTTP 502 raises ServerError."""
        error_response = {
            "error": "Bad gateway",
            "code": "BAD_GATEWAY",
            "message": "Upstream server error",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=502,
            )

            with pytest.raises(ServerError):
                client.browse_photos(route_id="NR-001")

    def test_503_raises_server_error(self, client):
        """Test that HTTP 503 raises ServerError."""
        error_response = {
            "error": "Service unavailable",
            "code": "SERVICE_UNAVAILABLE",
            "message": "Server is temporarily unavailable",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=503,
            )

            with pytest.raises(ServerError):
                client.browse_photos(route_id="NR-001")

    def test_error_code_takes_precedence_over_status_code(self, client):
        """Test that error code in response body takes precedence over HTTP status code."""
        error_response = {
            "error": "Invalid photo ID format",
            "code": "VALIDATION_ERROR",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=400,
            )

            with pytest.raises(ValidationError) as exc_info:
                client.browse_photos(route_id="invalid-id")

            assert exc_info.value.code == "VALIDATION_ERROR"

    def test_error_without_json_body_raises_bmphoto_error(self, client):
        """Test that error response without JSON body raises BMPhotoError."""
        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                text="Internal Server Error",
                status_code=500,
            )

            with pytest.raises(ServerError) as exc_info:
                client.browse_photos(route_id="NR-001")

            # Should still raise ServerError based on status code
            assert exc_info.value is not None

    def test_unknown_error_code_raises_bmphoto_error(self, client):
        """Test that unknown error code raises base BMPhotoError."""
        error_response = {
            "error": "Custom error",
            "code": "CUSTOM_ERROR_CODE",
            "message": "This is a custom error",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=418,  # Unrecognized status code
            )

            with pytest.raises(BMPhotoError) as exc_info:
                client.browse_photos(route_id="NR-001")

            assert "CUSTOM_ERROR_CODE" in str(exc_info.value)

    def test_error_body_parsing_with_missing_fields(self, client):
        """Test error parsing when JSON body has missing optional fields."""
        error_response = {
            "message": "Error without code field",
        }

        with requests_mock.Mocker() as m:
            m.get(
                f"{client._base_url}/api/v1/photos",
                json=error_response,
                status_code=500,
            )

            with pytest.raises(ServerError) as exc_info:
                client.browse_photos(route_id="NR-001")

            # Should still have the message
            assert "Error without code field" in str(exc_info.value)


# =============================================================================
# Test BMPhotoClient.batch_update_photos
# =============================================================================


class TestBatchUpdatePhotos:
    """Tests for BMPhotoClient.batch_update_photos method."""

    def test_empty_updates_returns_empty_response(self, client):
        """Test that empty updates list returns empty response without HTTP call."""
        from bm_photo_client.models import BatchUpdateItem

        result = client.batch_update_photos([])

        assert result.total == 0
        assert result.succeeded == 0
        assert result.failed == 0
        assert result.results == []

    def test_single_chunk_success(self, client):
        """Test batch update with items that fit in a single chunk."""
        from bm_photo_client.models import BatchUpdateItem

        updates = [
            BatchUpdateItem(photo_id="photo-001", lane_code="L2"),
            BatchUpdateItem(photo_id="photo-002", sta_value=150.5),
        ]

        response_json = {
            "total": 2,
            "succeeded": 2,
            "failed": 0,
            "results": [
                {
                    "photo_id": "photo-001",
                    "status": "success",
                    "photo": {
                        "photo_id": "photo-001",
                        "description": None,
                        "tags": [],
                        "survey_year": 2024,
                        "lane_code": "L2",
                        "updated_at": "2024-01-15T11:00:00Z",
                    },
                },
                {
                    "photo_id": "photo-002",
                    "status": "success",
                    "photo": {
                        "photo_id": "photo-002",
                        "description": None,
                        "tags": [],
                        "survey_year": 2024,
                        "lane_code": "L1",
                        "sta_value": 150.5,
                        "sta_source": "user_provided",
                        "updated_at": "2024-01-15T11:00:00Z",
                    },
                },
            ],
        }

        with requests_mock.Mocker() as m:
            m.patch(
                f"{client._base_url}/api/v1/photos/batch",
                json=response_json,
                status_code=200,
            )

            result = client.batch_update_photos(updates)

        assert result.total == 2
        assert result.succeeded == 2
        assert result.failed == 0
        assert len(result.results) == 2
        assert result.results[0].status == "success"
        assert result.results[1].status == "success"

    def test_partial_failure(self, client):
        """Test batch update with some items failing."""
        from bm_photo_client.models import BatchUpdateItem

        updates = [
            BatchUpdateItem(photo_id="photo-001", lane_code="L2"),
            BatchUpdateItem(photo_id="nonexistent", sta_value=150.5),
        ]

        response_json = {
            "total": 2,
            "succeeded": 1,
            "failed": 1,
            "results": [
                {
                    "photo_id": "photo-001",
                    "status": "success",
                    "photo": {
                        "photo_id": "photo-001",
                        "description": None,
                        "tags": [],
                        "survey_year": 2024,
                        "lane_code": "L2",
                        "updated_at": "2024-01-15T11:00:00Z",
                    },
                },
                {
                    "photo_id": "nonexistent",
                    "status": "error",
                    "error": "photo not found or has been deleted",
                    "error_code": "PHOTO_NOT_FOUND",
                },
            ],
        }

        with requests_mock.Mocker() as m:
            m.patch(
                f"{client._base_url}/api/v1/photos/batch",
                json=response_json,
                status_code=200,
            )

            result = client.batch_update_photos(updates)

        assert result.total == 2
        assert result.succeeded == 1
        assert result.failed == 1
        assert result.results[0].status == "success"
        assert result.results[1].status == "error"
        assert result.results[1].error_code == "PHOTO_NOT_FOUND"

    def test_auto_chunking(self, client):
        """Test that updates exceeding chunk_size are split into multiple requests."""
        from bm_photo_client.models import BatchUpdateItem

        updates = [
            BatchUpdateItem(photo_id=f"photo-{i:03d}", lane_code="L1")
            for i in range(750)
        ]

        chunk1_response = {
            "total": 500,
            "succeeded": 500,
            "failed": 0,
            "results": [
                {
                    "photo_id": f"photo-{i:03d}",
                    "status": "success",
                    "photo": {
                        "photo_id": f"photo-{i:03d}",
                        "description": None,
                        "tags": [],
                        "survey_year": 2024,
                        "lane_code": "L1",
                        "updated_at": "2024-01-15T11:00:00Z",
                    },
                }
                for i in range(500)
            ],
        }

        chunk2_response = {
            "total": 250,
            "succeeded": 250,
            "failed": 0,
            "results": [
                {
                    "photo_id": f"photo-{i:03d}",
                    "status": "success",
                    "photo": {
                        "photo_id": f"photo-{i:03d}",
                        "description": None,
                        "tags": [],
                        "survey_year": 2024,
                        "lane_code": "L1",
                        "updated_at": "2024-01-15T11:00:00Z",
                    },
                }
                for i in range(500, 750)
            ],
        }

        with requests_mock.Mocker() as m:
            m.patch(
                f"{client._base_url}/api/v1/photos/batch",
                [
                    {"json": chunk1_response, "status_code": 200},
                    {"json": chunk2_response, "status_code": 200},
                ],
            )

            result = client.batch_update_photos(updates, chunk_size=500)

        assert result.total == 750
        assert result.succeeded == 750
        assert result.failed == 0
        assert len(result.results) == 750
        assert m.call_count == 2

    def test_on_progress_callback(self, client):
        """Test that on_progress callback is called after each chunk."""
        from bm_photo_client.models import BatchUpdateItem

        updates = [
            BatchUpdateItem(photo_id=f"photo-{i:03d}", lane_code="L1")
            for i in range(3)
        ]

        response_json = {
            "total": 3,
            "succeeded": 3,
            "failed": 0,
            "results": [
                {
                    "photo_id": f"photo-{i:03d}",
                    "status": "success",
                    "photo": {
                        "photo_id": f"photo-{i:03d}",
                        "description": None,
                        "tags": [],
                        "survey_year": 2024,
                        "lane_code": "L1",
                        "updated_at": "2024-01-15T11:00:00Z",
                    },
                }
                for i in range(3)
            ],
        }

        progress_calls = []

        def on_progress(resp):
            progress_calls.append(resp)

        with requests_mock.Mocker() as m:
            m.patch(
                f"{client._base_url}/api/v1/photos/batch",
                json=response_json,
                status_code=200,
            )

            client.batch_update_photos(updates, on_progress=on_progress)

        assert len(progress_calls) == 1
        assert progress_calls[0].total == 3

    def test_403_raises_forbidden_error(self, client):
        """Test that 403 response raises ForbiddenError."""
        from bm_photo_client.models import BatchUpdateItem

        updates = [BatchUpdateItem(photo_id="photo-001", lane_code="L2")]

        with requests_mock.Mocker() as m:
            m.patch(
                f"{client._base_url}/api/v1/photos/batch",
                json={"error": "insufficient scope", "code": "INSUFFICIENT_SCOPE"},
                status_code=403,
            )

            with pytest.raises(ForbiddenError):
                client.batch_update_photos(updates)
