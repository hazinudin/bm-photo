"""Pydantic v2 models for Bina Marga Photo Service API responses."""

from datetime import datetime
from typing import List, Literal, Optional

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


class UpdatePhotoResponse(BaseModel):
    """Response from the PATCH /api/v1/photos/{photo_id} endpoint.

    Attributes:
        photo_id: Unique identifier for the photo.
        description: Updated description of the photo.
        tags: List of tags associated with the photo.
        survey_year: Year the survey was conducted.
        lane_code: Code identifying the lane.
        latitude: GPS latitude coordinate.
        longitude: GPS longitude coordinate.
        sta_value: Station value along the route in meters.
        sta_source: Source of the STA value (e.g., "user_provided").
        updated_at: Timestamp when the photo was last updated.
    """

    model_config = ConfigDict(populate_by_name=True)

    photo_id: str = Field(..., description="Unique identifier for the photo")
    description: Optional[str] = Field(None, description="Description of the photo")
    tags: List[str] = Field(
        default_factory=list, description="Tags associated with the photo"
    )
    survey_year: int = Field(..., description="Year the survey was conducted")
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
    updated_at: datetime = Field(
        ..., description="Timestamp when the photo was last updated"
    )


class BatchUpdateItem(BaseModel):
    """Single item in a batch update request.

    Attributes:
        photo_id: Unique identifier for the photo to update.
        description: New description for the photo.
        tags: New tags list for the photo.
        survey_year: New survey year.
        lane_code: New lane code (e.g., "L1", "R2").
        latitude: New GPS latitude coordinate.
        longitude: New GPS longitude coordinate.
        sta_value: New station value along the route.
    """

    model_config = ConfigDict(populate_by_name=True)

    photo_id: str = Field(..., description="Unique identifier for the photo to update")
    description: Optional[str] = Field(None, description="New description for the photo")
    tags: Optional[List[str]] = Field(None, description="New tags list")
    survey_year: Optional[int] = Field(None, description="New survey year")
    lane_code: Optional[str] = Field(None, description="New lane code")
    latitude: Optional[float] = Field(None, description="New latitude coordinate")
    longitude: Optional[float] = Field(None, description="New longitude coordinate")
    sta_value: Optional[float] = Field(None, description="New station value")


class BatchUpdateItemResult(BaseModel):
    """Result for a single item in a batch update response.

    Attributes:
        photo_id: Unique identifier for the photo.
        status: "success" or "error".
        error: Error message if status is "error".
        error_code: Machine-readable error code if status is "error".
        photo: Updated photo data if status is "success".
    """

    model_config = ConfigDict(populate_by_name=True)

    photo_id: str = Field(..., description="Unique identifier for the photo")
    status: Literal["success", "error"] = Field(..., description="Result status")
    error: Optional[str] = Field(None, description="Error message if failed")
    error_code: Optional[str] = Field(None, description="Error code if failed")
    photo: Optional[UpdatePhotoResponse] = Field(
        None, description="Updated photo data if succeeded"
    )


class BatchUpdateResponse(BaseModel):
    """Response from the PATCH /api/v1/photos/batch endpoint.

    Attributes:
        total: Total number of items in the batch.
        succeeded: Number of items that were successfully updated.
        failed: Number of items that failed to update.
        results: Per-item results.
    """

    model_config = ConfigDict(populate_by_name=True)

    total: int = Field(..., description="Total number of items in the batch")
    succeeded: int = Field(..., description="Number of successful updates")
    failed: int = Field(..., description="Number of failed updates")
    results: List[BatchUpdateItemResult] = Field(
        ..., description="Per-item results"
    )
