"""Exception classes for the BM Photo Client library.

This module defines a hierarchy of exceptions that map to the API error codes
and HTTP status codes returned by the BM Photo API service.

Example:
    >>> try:
    ...     client.photos.get("photo-123")
    ... except NotFoundError as e:
    ...     print(f"Photo not found: {e.code}")
"""

from typing import Optional


class BMPhotoError(Exception):
    """Base exception class for all BM Photo Client errors.

    All custom exceptions in this library inherit from BMPhotoError,
    providing a consistent interface for error handling.

    Attributes:
        error: Human-readable error message describing what went wrong.
        code: Machine-readable error code from the API (e.g., "PHOTO_NOT_FOUND").
        details: Additional context or debugging information, if available.

    Example:
        >>> try:
        ...     raise BMPhotoError("Something went wrong", "INTERNAL_ERROR", "DB timeout")
        ... except BMPhotoError as e:
        ...     print(e)
        Something went wrong (code=INTERNAL_ERROR)
    """

    def __init__(
        self,
        error: str,
        code: Optional[str] = None,
        details: Optional[str] = None,
    ) -> None:
        self.error = error
        self.code = code
        self.details = details
        super().__init__(error)

    def __str__(self) -> str:
        """Return a human-readable string representation of the error."""
        if self.code:
            return f"{self.error} (code={self.code})"
        return self.error

    def __repr__(self) -> str:
        """Return a detailed string representation for debugging."""
        return (
            f"{self.__class__.__name__}("
            f"error={self.error!r}, "
            f"code={self.code!r}, "
            f"details={self.details!r})"
        )


class AuthenticationError(BMPhotoError):
    """Raised when authentication fails due to missing, invalid, or expired credentials.

    This exception corresponds to HTTP 401 Unauthorized responses and covers
    the following error codes:
    - MISSING_API_KEY: No API key was provided in the request
    - INVALID_API_KEY: The provided API key is malformed or incorrect
    - INACTIVE_API_KEY: The API key has been deactivated
    - EXPIRED_API_KEY: The API key has expired

    Example:
        >>> raise AuthenticationError("API key has expired", "EXPIRED_API_KEY")
    """


class ForbiddenError(BMPhotoError):
    """Raised when the authenticated user lacks permission to perform an action.

    This exception corresponds to HTTP 403 Forbidden responses and covers
    the following error codes:
    - INSUFFICIENT_SCOPE: The API key lacks the required permissions
    - FORBIDDEN: Access to the resource is denied

    Example:
        >>> raise ForbiddenError("Insufficient permissions", "INSUFFICIENT_SCOPE")
    """


class NotFoundError(BMPhotoError):
    """Raised when the requested resource cannot be found.

    This exception corresponds to HTTP 404 Not Found responses and covers
    the following error codes:
    - NOT_FOUND: The requested resource does not exist
    - PHOTO_NOT_FOUND: The specified photo does not exist

    Example:
        >>> raise NotFoundError("Photo not found", "PHOTO_NOT_FOUND")
    """


class ValidationError(BMPhotoError):
    """Raised when request data fails validation.

    This exception corresponds to HTTP 400 Bad Request responses and covers
    the following error codes:
    - VALIDATION_ERROR: Input data failed validation checks
    - BAD_REQUEST: The request is malformed or invalid
    - INVALID_PHOTO_ID: The provided photo ID is not valid

    Example:
        >>> raise ValidationError("Invalid input", "VALIDATION_ERROR")
    """


class RateLimitError(BMPhotoError):
    """Raised when the client exceeds the allowed request rate or quota.

    This exception corresponds to HTTP 429 Too Many Requests responses and
    covers the following error codes:
    - RATE_LIMIT_EXCEEDED: Too many requests in a given time period
    - QUOTA_EXCEEDED: Monthly or usage quota has been exhausted

    Example:
        >>> raise RateLimitError("Rate limit exceeded", "RATE_LIMIT_EXCEEDED")
    """


class ServerError(BMPhotoError):
    """Raised when the server encounters an internal error.

    This exception corresponds to HTTP 5xx Server Error responses and covers
    the following error codes:
    - INTERNAL_ERROR: An unexpected error occurred on the server
    - Any generic 5xx status codes (500, 502, 503, etc.)

    Example:
        >>> raise ServerError("Internal server error", "INTERNAL_ERROR")
    """


# Mapping of HTTP status codes to exception classes
_STATUS_CODE_MAPPING: dict[int, type[BMPhotoError]] = {
    401: AuthenticationError,
    403: ForbiddenError,
    404: NotFoundError,
    400: ValidationError,
    429: RateLimitError,
    500: ServerError,
    502: ServerError,
    503: ServerError,
    504: ServerError,
}

# Mapping of error codes to exception classes
_ERROR_CODE_MAPPING: dict[str, type[BMPhotoError]] = {
    # Authentication errors (401)
    "MISSING_API_KEY": AuthenticationError,
    "INVALID_API_KEY": AuthenticationError,
    "INACTIVE_API_KEY": AuthenticationError,
    "EXPIRED_API_KEY": AuthenticationError,
    # Forbidden errors (403)
    "INSUFFICIENT_SCOPE": ForbiddenError,
    "FORBIDDEN": ForbiddenError,
    # Not found errors (404)
    "NOT_FOUND": NotFoundError,
    "PHOTO_NOT_FOUND": NotFoundError,
    # Validation errors (400)
    "VALIDATION_ERROR": ValidationError,
    "BAD_REQUEST": ValidationError,
    "INVALID_PHOTO_ID": ValidationError,
    # Rate limit errors (429)
    "RATE_LIMIT_EXCEEDED": RateLimitError,
    "QUOTA_EXCEEDED": RateLimitError,
    # Server errors (5xx)
    "INTERNAL_ERROR": ServerError,
}


def from_response(
    status_code: int, body: Optional[dict[str, str]] = None
) -> BMPhotoError:
    """Create an appropriate exception from an HTTP response.

    This factory function examines the HTTP status code and response body
    to determine the most specific exception type to raise.

    Args:
        status_code: The HTTP status code from the response.
        body: Optional dictionary containing error information.
            Expected format: {"error": "...", "code": "...", "details": "..."}
            If None or missing keys, defaults will be used.

    Returns:
        An instance of the appropriate exception class (BMPhotoError or a subclass).

    Example:
        >>> body = {"error": "Photo not found", "code": "PHOTO_NOT_FOUND"}
        >>> exc = from_response(404, body)
        >>> print(exc)
        Photo not found (code=PHOTO_NOT_FOUND)
        >>> isinstance(exc, NotFoundError)
        True

    Example with unrecognized error:
        >>> exc = from_response(500, {"error": "Something went wrong"})
        >>> isinstance(exc, ServerError)
        True
    """
    error = "An error occurred"
    code = None
    details = None

    if body:
        error = body.get("error", error)
        code = body.get("code")
        details = body.get("details")

    # First, try to match by error code if provided
    if code and code in _ERROR_CODE_MAPPING:
        exception_class = _ERROR_CODE_MAPPING[code]
        return exception_class(error=error, code=code, details=details)

    # Fall back to status code mapping
    if status_code in _STATUS_CODE_MAPPING:
        exception_class = _STATUS_CODE_MAPPING[status_code]
        return exception_class(error=error, code=code, details=details)

    # Default to base exception for unrecognized errors
    return BMPhotoError(error=error, code=code, details=details)
