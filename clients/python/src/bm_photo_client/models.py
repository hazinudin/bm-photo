"""Pydantic v2 models for Bina Marga Photo Service API responses."""

from datetime import datetime
from typing import List, Optional

from pydantic import BaseModel, ConfigDict, Field


class PhotoSummary(BaseModel):
    """Summary representation of a photo from the browse photos endpoint.

    Attributes:
        photo_id: Unique identifier for the photo.
        route_id: The national route identifier (e.g., "NR-001").
        lane_code: Code identifying the lane (e.g., "L1", "L2").
        sta_value: Station value along the route in meters.
        survey_year: Year the survey was conducted.
        gcs_url: Google Cloud Storage URL for the photo.
        uploaded_at: Timestamp when the photo was uploaded.
        file_name: Original file name of the photo.
    """

    model_config = ConfigDict(populate_by_name=True)

    photo_id: str = Field(..., description="Unique identifier for the photo")
    route_id: str = Field(..., description="The national route identifier")
    lane_code: str = Field(..., description="Code identifying the lane")
    sta_value: Optional[float] = Field(
        None, description="Station value along the route in meters"
    )
    survey_year: int = Field(..., description="Year the survey was conducted")
    gcs_url: str = Field(..., description="Google Cloud Storage URL for the photo")
    uploaded_at: datetime = Field(
        ..., description="Timestamp when the photo was uploaded"
    )
    file_name: str = Field(..., description="Original file name of the photo")


class Pagination(BaseModel):
    """Pagination metadata for list responses.

    Attributes:
        current_page: The current page number (1-indexed).
        per_page: Number of items per page.
        total_count: Total number of items across all pages.
        total_pages: Total number of pages available.
    """

    model_config = ConfigDict(populate_by_name=True)

    current_page: int = Field(..., ge=1, description="Current page number (1-indexed)")
    per_page: int = Field(..., ge=1, description="Number of items per page")
    total_count: int = Field(..., ge=0, description="Total number of items")
    total_pages: int = Field(..., ge=0, description="Total number of pages")


class BrowsePhotosResponse(BaseModel):
    """Response from the GET /api/v1/photos endpoint.

    Contains a list of photo summaries and pagination metadata.

    Attributes:
        photos: List of photo summaries.
        pagination: Pagination metadata for the response.
    """

    model_config = ConfigDict(populate_by_name=True)

    photos: List[PhotoSummary] = Field(
        ..., description="List of photo summaries on this page"
    )
    pagination: Pagination = Field(..., description="Pagination metadata")


class PhotoDetail(BaseModel):
    """Detailed representation of a photo from the get photo endpoint.

    Attributes:
        photo_id: Unique identifier for the photo.
        route_id: The national route identifier (e.g., "NR-001").
        lane_code: Code identifying the lane (e.g., "L1", "L2").
        latitude: GPS latitude coordinate of the photo location.
        longitude: GPS longitude coordinate of the photo location.
        sta_value: Station value along the route in meters.
        sta_source: Source of the STA value (e.g., "GPS", "Manual").
        file_format: File format/extension (e.g., "jpg", "png").
        file_size_bytes: Size of the file in bytes.
        description: Optional description of the photo.
        tags: List of tags associated with the photo.
        uploaded_at: Timestamp when the photo was uploaded.
        download_url: Signed URL to download the photo.
    """

    model_config = ConfigDict(populate_by_name=True)

    photo_id: str = Field(..., description="Unique identifier for the photo")
    route_id: str = Field(..., description="The national route identifier")
    lane_code: str = Field(..., description="Code identifying the lane")
    latitude: Optional[float] = Field(
        None,
        ge=-90.0,
        le=90.0,
        description="GPS latitude coordinate",
    )
    longitude: Optional[float] = Field(
        None,
        ge=-180.0,
        le=180.0,
        description="GPS longitude coordinate",
    )
    sta_value: Optional[float] = Field(
        None, description="Station value along the route in meters"
    )
    sta_source: Optional[str] = Field(None, description="Source of the STA value")
    file_format: str = Field(..., description="File format/extension")
    file_size_bytes: int = Field(..., ge=0, description="File size in bytes")
    description: Optional[str] = Field(
        None, description="Optional description of the photo"
    )
    tags: List[str] = Field(
        default_factory=list, description="Tags associated with the photo"
    )
    uploaded_at: datetime = Field(
        ..., description="Timestamp when the photo was uploaded"
    )
    download_url: str = Field(..., description="Signed URL to download the photo")
