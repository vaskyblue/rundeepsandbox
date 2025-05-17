# DeepSandbox Go API

A secure and scalable API built in Go for executing Python data analysis on client datasets. This is a Golang implementation of the DeepSandbox API.

## Features

- User Authentication & Authorization
- Dataset Management
- Secure Code Execution
- Rate Limiting
- Execution Quota Management

## API Endpoints

### Authentication

- `POST /api/v1/auth/token` - Get access token (login)
- `POST /api/v1/auth/register` - Register a new user
- `GET /api/v1/auth/users/me` - Get current user information
- `PUT /api/v1/auth/users/me` - Update current user information
- `GET /api/v1/auth/admin/users` - List all users (admin only)

### Datasets

- `POST /api/v1/datasets/upload` - Upload a new dataset
- `GET /api/v1/datasets` - List available datasets
- `GET /api/v1/datasets/{dataset_id}` - Get dataset information and preview
- `DELETE /api/v1/datasets/{dataset_id}` - Delete a dataset

### Code Execution

- `POST /api/v1/execute` - Submit code for execution
- `GET /api/v1/tasks/{task_id}` - Check task status
- `DELETE /api/v1/tasks/{task_id}` - Cancel a task
- `GET /api/v1/admin/queue-status` - Get queue status (admin only)

## Setup

### Prerequisites

- Go 1.20 or higher
- Docker and Docker Compose
- PostgreSQL
- Redis

### Running Locally

1. Clone the repository
2. Create a `.env` file with environment variables (see `.env.example`)
3. Run the application:

```bash
go mod download
go run main.go
```

### Running with Docker

```bash
# Build and run all services
docker-compose up -d

# View logs
docker-compose logs -f api

# Stop services
docker-compose down
```

## Configuration

Configuration can be set using environment variables or a `.env` file:

- `POSTGRES_HOST` - PostgreSQL host
- `POSTGRES_PORT` - PostgreSQL port
- `POSTGRES_USER` - PostgreSQL username
- `POSTGRES_PASSWORD` - PostgreSQL password
- `POSTGRES_DB` - PostgreSQL database name
- `REDIS_HOST` - Redis host
- `REDIS_PORT` - Redis port
- `SECRET_KEY` - Secret key for JWT tokens
- `ACCESS_TOKEN_EXPIRE_MINUTES` - JWT token expiration time in minutes
- `RATE_LIMIT_WINDOW` - Rate limit window in seconds
- `MAX_REQUESTS_PER_WINDOW` - Maximum requests per window
- `MAX_EXECUTIONS_PER_DAY` - Maximum code executions per day
- `CONTAINER_TIMEOUT` - Maximum execution time in seconds
- `DATASETS_DIR` - Directory to store datasets
- `API_TITLE` - API title
- `API_DESCRIPTION` - API description
- `API_VERSION` - API version
- `SERVER_PORT` - Server port

## License

MIT 

 docker build -t go-deepsandbox .

 docker run -p 8080:8080 go-deepsandbox