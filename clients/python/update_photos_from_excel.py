#!/usr/bin/env python3
"""Update photo metadata from Excel file using batch update API."""

import logging
import sys
import time
from pathlib import Path

import polars as pl

sys.path.insert(0, str(Path(__file__).parent / "src"))

from bm_photo_client import BMPhotoClient, BatchUpdateItem

EXCEL_FILE = Path(__file__).parent.parent.parent / "defect_14_07-06-2026_055405_5477.xlsx"
API_BASE = "http://localhost:8082"
API_KEY = "test-admin-api-key-12345678901234567890123456789012345678901234567890123456"

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger(__name__)


def main():
    logger.info(f"Loading {EXCEL_FILE} with Polars...")
    start_load = time.time()
    
    df = pl.read_excel(EXCEL_FILE)
    
    load_time = time.time() - start_load
    logger.info(f"Loaded {len(df)} rows in {load_time:.2f}s")
    
    # Filter out rows without photo_id
    df = df.filter(pl.col("photo_id").is_not_null())
    logger.info(f"After filtering null photo_id: {len(df)} rows")
    
    # Convert to updates
    updates = []
    skipped = 0
    
    for row in df.iter_rows(named=True):
        try:
            sta_val = float(row["sta"]) if row["sta"] is not None else None
            lat_val = float(row["sta_lat"]) if row["sta_lat"] is not None else None
            lon_val = float(row["sta_long"]) if row["sta_long"] is not None else None
        except (ValueError, TypeError):
            skipped += 1
            continue
        
        updates.append(
            BatchUpdateItem(
                photo_id=row["photo_id"],
                sta_value=sta_val,
                latitude=lat_val,
                longitude=lon_val,
            )
        )
    
    logger.info(f"Prepared {len(updates)} updates ({skipped} skipped due to conversion errors)")
    logger.info(f"Sending to {API_BASE}...")

    client = BMPhotoClient(base_url=API_BASE, api_key=API_KEY, timeout=120.0)

    total_items = len(updates)
    chunk_size = 200
    total_chunks = (total_items + chunk_size - 1) // chunk_size
    processed = 0
    start_time = time.time()

    def on_progress(chunk_response):
        nonlocal processed
        processed += chunk_response.total
        chunk_num = (processed - 1) // chunk_size + 1
        elapsed = time.time() - start_time
        rate = processed / elapsed if elapsed > 0 else 0
        eta = (total_items - processed) / rate if rate > 0 else 0
        
        logger.info(
            f"Chunk {chunk_num}/{total_chunks}: "
            f"{chunk_response.succeeded} succeeded, {chunk_response.failed} failed | "
            f"Total: {processed}/{total_items} ({processed/total_items*100:.1f}%) | "
            f"Rate: {rate:.1f} items/s | ETA: {eta:.0f}s"
        )

    logger.info(f"Processing {total_items} items in {total_chunks} batches of {chunk_size}...")
    result = client.batch_update_photos(updates, chunk_size=chunk_size, on_progress=on_progress)

    elapsed = time.time() - start_time
    logger.info("=" * 70)
    logger.info(f"COMPLETED in {elapsed:.1f}s")
    logger.info(f"Total:     {result.total}")
    logger.info(f"Succeeded: {result.succeeded}")
    logger.info(f"Failed:    {result.failed}")

    if result.failed > 0:
        logger.warning(f"Failed items ({result.failed}):")
        for item in result.results:
            if item.status == "error":
                logger.warning(f"  {item.photo_id}: {item.error_code} - {item.error}")


if __name__ == "__main__":
    main()
