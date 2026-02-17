# Initial Sync Feature Guide

This guide explains how to use the initial sync feature to synchronize existing data before starting CDC (Change Data Capture).

## Overview

The initial sync feature allows you to:
- **Sync existing data** from MongoDB to PostgreSQL before starting real-time CDC
- **Smart incremental sync** based on timestamp fields
- **Automatic detection** of whether sink is empty or has data
- **Force full resync** when needed
- **Configurable batch sizes** for performance tuning

## Configuration

Add the `sync` section to your pipeline configuration:

```json
{
  "pipeline": {
    "name": "my-pipeline",
    "sync": {
      "initial_sync": true,
      "force_initial_sync": false,
      "timestamp_field": "updated_at",
      "batch_size": 1000
    }
  },
  "source": { ... },
  "sink": { ... }
}
```

### Configuration Options

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `initial_sync` | boolean | Yes | `false` | Enable initial synchronization |
| `force_initial_sync` | boolean | No | `false` | Force full sync even if data exists in sink |
| `timestamp_field` | string | No | `""` | Field name to use for timestamp-based sync |
| `batch_size` | integer | No | `1000` | Number of documents per batch |

## How It Works

### Sync Logic Flow

```
┌─────────────────────────────────────────┐
│ Initial Sync Enabled?                   │
└────────────┬────────────────────────────┘
             │ Yes
             ▼
┌─────────────────────────────────────────┐
│ Force Initial Sync?                     │
└────────────┬────────────────────────────┘
             │ No
             ▼
┌─────────────────────────────────────────┐
│ Check if Sink Table is Empty            │
└────────────┬────────────────────────────┘
             │
        ┌────┴────┐
        │ Empty?  │
        └────┬────┘
   Yes       │       No
        ┌────▼────┐
        │  Full   │
        │  Sync   │
        └─────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Timestamp Field Configured?             │
└────────────┬────────────────────────────┘
             │ Yes
             ▼
┌─────────────────────────────────────────┐
│ Get Latest Timestamp from Sink          │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Sync Documents WHERE timestamp >= max   │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Start CDC for Real-time Changes         │
└─────────────────────────────────────────┘
```

### Scenario 1: Empty Sink (Full Sync)

**When**: Sink table has no data
**What happens**: All documents from MongoDB are synced to PostgreSQL

```json
{
  "sync": {
    "initial_sync": true,
    "timestamp_field": "updated_at"
  }
}
```

**Behavior**:
1. Pipeline checks if PostgreSQL table is empty
2. Table is empty → Full sync initiated
3. All documents from MongoDB collection are copied
4. After sync completes, CDC starts monitoring for new changes

**Log Output**:
```
[data-pipe] Initial sync is enabled
[data-pipe] Sink table is empty, performing full initial sync
[data-pipe] Starting full initial sync for mydb.users
[data-pipe] Initial sync progress: 1000 documents synced
[data-pipe] Initial sync progress: 2000 documents synced
[data-pipe] Initial sync completed: 2500 documents synced
[data-pipe] Starting CDC pipeline...
```

### Scenario 2: Incremental Sync (Timestamp-Based)

**When**: Sink has data and timestamp field is configured
**What happens**: Only documents newer than the latest timestamp in sink are synced

```json
{
  "sync": {
    "initial_sync": true,
    "timestamp_field": "updated_at"
  }
}
```

**Behavior**:
1. Pipeline checks PostgreSQL table - not empty
2. Queries PostgreSQL for latest `updated_at` timestamp
3. Queries MongoDB for documents where `updated_at >= latest_timestamp`
4. Syncs only those newer documents
5. Starts CDC for ongoing changes

**Log Output**:
```
[data-pipe] Initial sync is enabled
[data-pipe] Starting incremental initial sync from timestamp: 2024-02-15T10:30:00Z
[data-pipe] Initial sync completed: 150 documents synced
[data-pipe] Starting CDC pipeline...
```

### Scenario 3: Force Full Resync

**When**: You want to resync all data regardless of what's in the sink
**What happens**: All documents are synced, potentially overwriting existing data

```json
{
  "sync": {
    "initial_sync": true,
    "force_initial_sync": true,
    "timestamp_field": "updated_at"
  }
}
```

**Behavior**:
1. `force_initial_sync` is true → Skip sink checks
2. Sync all documents from MongoDB
3. Use upsert logic to update existing records
4. Start CDC for ongoing changes

**Log Output**:
```
[data-pipe] Initial sync is enabled
[data-pipe] Force initial sync is enabled, syncing all data
[data-pipe] Starting full initial sync for mydb.users
[data-pipe] Initial sync completed: 2500 documents synced
[data-pipe] Starting CDC pipeline...
```

### Scenario 4: CDC Only (No Initial Sync)

**When**: Initial sync is disabled
**What happens**: Only monitors for new changes, no historical data sync

```json
{
  "sync": {
    "initial_sync": false
  }
}
```

**Behavior**:
1. No initial sync performed
2. CDC starts immediately
3. Only changes after pipeline start are captured

## Requirements

### MongoDB Requirements

For timestamp-based sync, your MongoDB documents should have a timestamp field:

```javascript
// Good: Has timestamp field
{
  "_id": ObjectId("..."),
  "name": "John Doe",
  "email": "john@example.com",
  "created_at": ISODate("2024-01-15T10:00:00Z"),
  "updated_at": ISODate("2024-02-15T14:30:00Z")
}
```

The timestamp field can be:
- `Date` type (ISODate)
- `Timestamp` type
- Numeric timestamp (epoch milliseconds)
- String in ISO format

### PostgreSQL Requirements

Your PostgreSQL table should have the corresponding timestamp column:

```sql
CREATE TABLE users (
    _id TEXT PRIMARY KEY,
    name TEXT,
    email TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP  -- Must match timestamp_field in config
);

-- Optional: Add index for better performance
CREATE INDEX idx_users_updated_at ON users(updated_at);
```

## Performance Tuning

### Batch Size

Control how many documents are processed at once:

```json
{
  "sync": {
    "initial_sync": true,
    "batch_size": 5000  // Larger batches = faster sync, more memory
  }
}
```

**Recommendations**:
- **Small datasets** (<10K docs): 1000-2000
- **Medium datasets** (10K-1M docs): 2000-5000
- **Large datasets** (>1M docs): 5000-10000

**Trade-offs**:
- Larger batches: Faster sync, higher memory usage
- Smaller batches: Slower sync, lower memory usage, more granular progress

### Indexing

Add indexes on timestamp fields for better performance:

```sql
-- PostgreSQL
CREATE INDEX idx_users_updated_at ON users(updated_at DESC);

-- MongoDB (if not already indexed)
db.users.createIndex({ "updated_at": 1 })
```

## Common Patterns

### Pattern 1: Daily Full Resync

Run a full resync once per day:

```bash
# Daily cron job
0 2 * * * /path/to/data-pipe -config /path/to/config-force-sync.json
```

```json
{
  "sync": {
    "initial_sync": true,
    "force_initial_sync": true
  }
}
```

### Pattern 2: Catch-Up Sync

Resume syncing after downtime using timestamp:

```json
{
  "sync": {
    "initial_sync": true,
    "timestamp_field": "updated_at"
  }
}
```

When pipeline restarts, it automatically syncs documents modified during downtime.

### Pattern 3: One-Time Historical Load + CDC

First run with initial sync, subsequent runs without:

```bash
# First run: Load all historical data
./data-pipe -config config-with-initial-sync.json

# Stop and update config (set initial_sync: false)

# Subsequent runs: CDC only
./data-pipe -config config-cdc-only.json
```

## Troubleshooting

### Issue: Initial sync is slow

**Solution**:
1. Increase `batch_size` to 5000 or 10000
2. Add indexes on timestamp fields
3. Ensure network latency is low between MongoDB and PostgreSQL

### Issue: Duplicate data in sink

**Cause**: Running with `force_initial_sync: true` multiple times

**Solution**:
- The pipeline uses upsert logic, so duplicates should be prevented
- Ensure your table has a proper primary key on `_id`
- Check for unique constraints

### Issue: Missing documents after initial sync

**Cause**: Documents modified during initial sync might be missed

**Solution**:
- Use timestamp-based sync with a safety margin
- Ensure CDC starts immediately after initial sync completes
- The pipeline is designed to handle this seamlessly

### Issue: "Timestamp field not found"

**Error**: `timestamp field 'updated_at' not found in document`

**Solution**:
1. Verify all documents have the timestamp field
2. Check field name matches exactly (case-sensitive)
3. Consider adding the field to existing documents:

```javascript
// MongoDB
db.users.updateMany(
  { updated_at: { $exists: false } },
  { $set: { updated_at: new Date() } }
)
```

## Best Practices

1. **Always use timestamp fields** for production deployments
2. **Test with small datasets** before running on production data
3. **Monitor initial sync progress** in logs
4. **Use appropriate batch sizes** for your data volume
5. **Add indexes** on timestamp fields for better performance
6. **Run initial sync during low-traffic periods** for large datasets
7. **Keep `force_initial_sync: false`** for normal operations
8. **Document your sync strategy** for your team

## Examples

See the `examples/` directory for complete configuration examples:
- `config-with-initial-sync.json` - Full configuration with initial sync
- `config.json` - Basic CDC-only configuration

## Comparison: Initial Sync vs CDC Only

| Feature | Initial Sync Enabled | CDC Only |
|---------|---------------------|----------|
| Syncs existing data | ✅ Yes | ❌ No |
| Syncs new changes | ✅ Yes | ✅ Yes |
| Startup time | Slower (depends on data volume) | Fast |
| Use case | Fresh deployment, catch-up | Already in sync |
| Configuration | Requires timestamp field | None |

## Summary

The initial sync feature provides a complete solution for:
- **First-time deployment**: Sync all historical data
- **Recovery from downtime**: Catch up on missed changes
- **Periodic resyncs**: Force full data refresh
- **Incremental updates**: Sync only new/modified data

Enable initial sync in your configuration, and the pipeline handles the rest automatically!
