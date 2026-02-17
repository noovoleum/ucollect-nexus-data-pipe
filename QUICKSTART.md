# Quick Start Guide

This guide will help you get started with the data-pipe CDC pipeline in minutes.

## Prerequisites

- Docker and Docker Compose (recommended)
- OR Go 1.20+ if building from source

## Option 1: Quick Start with Docker Compose (Recommended)

This will set up MongoDB, PostgreSQL, and the data pipeline in one command.

### Step 1: Clone the Repository

```bash
git clone https://github.com/IEatCodeDaily/data-pipe.git
cd data-pipe
```

### Step 2: Create Configuration

Copy the example configuration:

```bash
cp examples/config.json config.json
```

Edit `config.json` to match your needs (the default works with docker-compose):

```json
{
  "pipeline": {
    "name": "mongodb-to-postgresql"
  },
  "source": {
    "type": "mongodb",
    "settings": {
      "uri": "mongodb://admin:password@mongodb:27017/?replicaSet=rs0",
      "database": "mydb",
      "collection": "users"
    }
  },
  "sink": {
    "type": "postgresql",
    "settings": {
      "connection_string": "host=postgres port=5432 user=postgres password=postgres dbname=mydb sslmode=disable",
      "table": "users"
    }
  }
}
```

### Step 3: Start Services

```bash
docker-compose up -d
```

This starts:
- MongoDB with replica set
- PostgreSQL
- data-pipe (will wait for MongoDB and PostgreSQL)

### Step 4: Initialize MongoDB Replica Set

MongoDB change streams require a replica set. Initialize it:

```bash
docker exec -it mongodb mongosh --eval "rs.initiate()"
```

Wait a few seconds for the replica set to initialize, then restart the data-pipe:

```bash
docker-compose restart data-pipe
```

### Step 5: Set Up PostgreSQL Table

Create the target table in PostgreSQL:

```bash
docker exec -it postgres psql -U postgres -d mydb -c "
CREATE TABLE users (
    _id TEXT PRIMARY KEY,
    name TEXT,
    email TEXT,
    created_at TIMESTAMP
);
"
```

### Step 6: Test the Pipeline

**Important Note**: The pipeline only syncs **new changes** after it starts. It does NOT sync existing data. See the "Understanding CDC Mode" section below.

Insert a document in MongoDB:

```bash
docker exec -it mongodb mongosh --eval "
use mydb
db.users.insertOne({
  name: 'Alice Smith',
  email: 'alice@example.com',
  created_at: new Date()
})
"
```

Check if it synced to PostgreSQL:

```bash
docker exec -it postgres psql -U postgres -d mydb -c "SELECT * FROM users;"
```

You should see the record!

### Step 7: Monitor Logs

```bash
# Watch data-pipe logs
docker-compose logs -f data-pipe

# Or view all logs
docker-compose logs -f
```

### Stop Services

```bash
docker-compose down
```

## Option 2: Build from Source

### Step 1: Install Dependencies

```bash
# Install Go 1.20 or higher
# Then clone the repository
git clone https://github.com/IEatCodeDaily/data-pipe.git
cd data-pipe
```

### Step 2: Build

```bash
make build
# Or: go build -o data-pipe ./cmd/data-pipe
```

### Step 3: Set Up Databases

You need MongoDB (with replica set) and PostgreSQL running. Use docker-compose for databases only:

```yaml
# docker-compose-db.yml
services:
  mongodb:
    image: mongo:7
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: password
    command: mongod --replSet rs0

  postgres:
    image: postgres:16
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: mydb
```

```bash
docker-compose -f docker-compose-db.yml up -d
docker exec -it mongodb mongosh --eval "rs.initiate()"
```

### Step 4: Configure

Create `config.json`:

```json
{
  "pipeline": {
    "name": "mongodb-to-postgresql"
  },
  "source": {
    "type": "mongodb",
    "settings": {
      "uri": "mongodb://admin:password@localhost:27017/?replicaSet=rs0",
      "database": "mydb",
      "collection": "users"
    }
  },
  "sink": {
    "type": "postgresql",
    "settings": {
      "connection_string": "host=localhost port=5432 user=postgres password=postgres dbname=mydb sslmode=disable",
      "table": "users"
    }
  }
}
```

### Step 5: Run

```bash
./data-pipe -config config.json
```

## Testing the Pipeline

### Insert Operations

```javascript
// MongoDB
db.users.insertOne({
  name: "John Doe",
  email: "john@example.com",
  age: 30
})
```

### Update Operations

```javascript
// MongoDB
db.users.updateOne(
  { email: "john@example.com" },
  { $set: { age: 31 } }
)
```

### Delete Operations

```javascript
// MongoDB
db.users.deleteOne({ email: "john@example.com" })
```

All changes will automatically sync to PostgreSQL!

## Initial Sync Feature (New!)

The pipeline now supports automatic initial synchronization of existing data:

### Quick Enable

Add to your `config.json`:

```json
{
  "pipeline": {
    "name": "mongodb-to-postgresql",
    "sync": {
      "initial_sync": true,
      "timestamp_field": "updated_at",
      "batch_size": 1000
    }
  }
}
```

### How It Works

1. **Empty sink**: Syncs all existing MongoDB data
2. **Has data**: Syncs only documents newer than latest timestamp in sink
3. **After sync**: Starts CDC for real-time changes

See `INITIAL_SYNC.md` for complete documentation.

## Understanding CDC Mode

**With Initial Sync Enabled** (Recommended):
- ✅ Syncs existing documents first
- ✅ Then monitors for new changes
- ✅ No data loss on first deployment

**With Initial Sync Disabled** (Legacy):

**Important**: The pipeline operates in **Change Data Capture (CDC) mode only** when initial sync is disabled:

### What Gets Synced
- ✅ Documents created **after** the pipeline starts
- ✅ Documents updated **after** the pipeline starts
- ✅ Documents deleted **after** the pipeline starts

### What Does NOT Get Synced
- ❌ Existing documents in MongoDB before pipeline start
- ❌ Historical data (no backfill)

### For Initial Data Sync

If you have existing data in MongoDB that needs to be in PostgreSQL:

#### Option 1: Manual Export/Import

```bash
# Export from MongoDB
mongoexport --uri="mongodb://localhost:27017" --db=mydb --collection=users --out=users.json

# Import to PostgreSQL (write a custom script)
# Then start data-pipe for ongoing sync
```

#### Option 2: Copy Existing Data First

```bash
# Use mongosh to copy existing documents
docker exec -it mongodb mongosh --eval "
use mydb
db.users.find().forEach(function(doc) {
  // Manually insert into PostgreSQL first
})
"

# Then start the pipeline
./data-pipe -config config.json
```

#### Option 3: Start Fresh

```bash
# Clear both databases and start with new data
# All new operations will be synced automatically
```

This is a common CDC pattern - you do a one-time bulk load of existing data, then use CDC for ongoing synchronization.

## Common Issues

### Change Stream Not Working

**Problem**: `stream returned error: (Location40573) The $changeStream stage is only supported on replica sets`

**Solution**: Ensure MongoDB is running as a replica set and initialized:
```bash
docker exec -it mongodb mongosh --eval "rs.initiate()"
```

### Connection Refused

**Problem**: `connection refused` errors

**Solution**: 
- Ensure services are running: `docker-compose ps`
- Check logs: `docker-compose logs mongodb postgres`
- Give services time to start (10-15 seconds)

### Table Not Found

**Problem**: PostgreSQL errors about missing table

**Solution**: Create the table first:
```sql
CREATE TABLE users (
    _id TEXT PRIMARY KEY,
    -- add other fields as needed
);
```

## Next Steps

- Read [README.md](README.md) for detailed documentation
- See [EXTENDING.md](EXTENDING.md) for adding new connectors
- Check configuration examples in `examples/`

## Production Considerations

Before deploying to production:

1. **Secure Credentials**: Use environment variables or secrets management
2. **Monitoring**: Add Prometheus/Grafana for metrics
3. **Logging**: Configure structured logging
4. **High Availability**: Run multiple instances with load balancing
5. **Backups**: Ensure both databases have backup strategies
6. **Testing**: Test with production-like data volumes

## Support

For issues and questions:
- Open an issue on GitHub
- Check existing documentation
- Review example configurations
