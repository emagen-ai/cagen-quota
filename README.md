# Cagen Quota Service

A distributed quota management service for the Cagen ecosystem, providing hierarchical quota allocation and usage tracking with integrated permission management.

## Features

- **Hierarchical Quota Management**: Support for organization-level and team-level quotas
- **Permission Integration**: Seamless integration with Cagen Auth Service
- **Usage Tracking**: Real-time quota usage monitoring and allocation
- **Audit Logging**: Complete audit trail for all quota operations
- **Organization Isolation**: Strict isolation between organizations
- **RESTful API**: Clean HTTP API for all operations

## Architecture

```
┌─────────────────┐    权限验证    ┌─────────────────────┐
│   业务服务       │ ←─────────→   │  Cagen-Auth-Service │
│  (quota请求)    │               │                     │
└─────────────────┘               └─────────────────────┘
         ↓                                  ↑
         ↓                                  ↑ 权限检查
┌─────────────────┐                         ↑
│ Cagen-Quota     │ ←───────────────────────┘
│ Service         │
│                 │
│ - 配额管理       │
│ - 配额分配       │
│ - 使用监控       │
└─────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL 12+
- Docker (optional)

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
# Database
DATABASE_URL=postgresql://username:password@localhost:5432/cagen_quota?sslmode=disable

# Server
PORT=8080
ENVIRONMENT=development

# Auth Service Integration
AUTH_SERVICE_URL=https://cagen-auth-service-production.up.railway.app
CAGEN_QUOTA_SERVICE_SECRET_KEY=your-32-byte-base64-key
QUOTA_SERVICE_ID=svc_cagen_quota
```

### Local Development

```bash
# Install dependencies
go mod download

# Run the service
go run main.go
```

### Docker Development

```bash
# Start all services
docker-compose up

# Stop services
docker-compose down
```

## API Documentation

### Core Endpoints

#### Health Check
```http
GET /health
```

#### Create Root Quota
```http
POST /api/v1/quotas/create
Content-Type: application/json

{
  "service_id": "svc_cagen_quota",
  "encrypted_data": "base64-encrypted-user-info",
  "name": "Organization Main Quota",
  "description": "Main quota for organization",
  "type": "organization",
  "total_mb": 10000
}
```

#### Allocate Sub-Quota
```http
POST /api/v1/quotas/{parent_quota_id}/allocate
Content-Type: application/json

{
  "service_id": "svc_cagen_quota",
  "encrypted_data": "base64-encrypted-user-info",
  "name": "Team Development Quota",
  "description": "Quota for development team",
  "allocate_mb": 2000,
  "type": "team",
  "target_id": "team_dev",
  "admin_user_ids": ["user_123", "user_456"]
}
```

#### Get Quota Details
```http
GET /api/v1/quotas/{quota_id}?service_id=svc_cagen_quota&encrypted_data=base64-data
```

#### Allocate Usage
```http
POST /api/v1/quotas/{quota_id}/usage/allocate
Content-Type: application/json

{
  "service_id": "svc_cagen_quota",
  "encrypted_data": "base64-encrypted-user-info",
  "resource_id": "resource_123",
  "usage_mb": 100,
  "reason": "File storage allocation"
}
```

#### Grant Permissions
```http
POST /api/v1/quotas/{quota_id}/permissions/grant
Content-Type: application/json

{
  "service_id": "svc_cagen_quota",
  "encrypted_data": "base64-encrypted-user-info",
  "target_user_id": "user_789",
  "permissions": ["read", "admin"]
}
```

## Permission Model

### Permission Types
- **read**: View quota information, use quota
- **admin**: Allocate sub-quotas, manage permissions
- **owner**: Full control (inherited from parent quota)

### Hierarchy Rules
- Organization quotas can allocate to team quotas
- Team quotas can only allocate within the same team
- Owner permissions are inherited down the hierarchy

## Quota States

- **active**: Normal operational state
- **suspended**: Temporarily disabled
- **deleted**: Soft-deleted (capacity returned to parent)

## Database Schema

### Core Tables

- `quotas`: Main quota records with hierarchy and capacity
- `quota_usage`: Usage tracking records
- `quota_audit_logs`: Complete audit trail

### Key Constraints

- Capacity balance: `used_mb + allocated_mb <= total_mb`
- Hierarchy integrity: Proper parent-child relationships
- Organization isolation: Strict data separation

## Development

### Project Structure

```
cagen-quota/
├── cmd/                    # Application entry points
├── internal/
│   ├── auth/              # Auth service client
│   ├── config/            # Configuration management
│   ├── database/          # Database connection and schema
│   ├── handlers/          # HTTP handlers
│   ├── models/            # Data models
│   └── services/          # Business logic
├── migrations/            # Database migrations
├── docs/                 # Documentation
└── scripts/              # Utility scripts
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...
```

### Database Migrations

```bash
# Apply migrations manually
psql $DATABASE_URL -f migrations/001_initial_schema.sql
```

## Deployment

### Railway Deployment

1. Create new Railway project
2. Connect GitHub repository
3. Set environment variables:
   - `DATABASE_URL`: PostgreSQL connection string
   - `CAGEN_QUOTA_SERVICE_SECRET_KEY`: 32-byte base64 key
   - `AUTH_SERVICE_URL`: Auth service endpoint
   - `ENVIRONMENT`: `production`
   - `LOG_FORMAT`: `json`

### Docker Deployment

```bash
# Build image
docker build -t cagen-quota .

# Run container
docker run -p 8080:8080 \
  -e DATABASE_URL="postgresql://..." \
  -e CAGEN_QUOTA_SERVICE_SECRET_KEY="..." \
  cagen-quota
```

## Monitoring

### Health Checks

The service provides health check endpoints:

- `GET /health`: Basic service health
- `GET /dev/info`: Development information (dev mode only)

### Logging

Structured logging with configurable levels:

- `LOG_LEVEL`: debug, info, warn, error
- `LOG_FORMAT`: text, json

### Metrics

Key metrics to monitor:

- Quota utilization rates
- API response times
- Error rates
- Database connection health

## Security

### Encryption

- All user data encrypted with AES-256-GCM
- Unique nonces prevent replay attacks
- Time-based validation (±5 minutes)

### Organization Isolation

- Strict database-level isolation
- User data scoped to organizations
- Cross-organization access blocked

### Audit Trail

- Complete operation logging
- Actor and target tracking
- Immutable audit records

## Contributing

1. Fork the repository
2. Create feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit pull request

## License

Private - Emagen AI Internal Use Only