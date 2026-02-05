
# Mass-Translate Server
ths is an API server wrapper to the mass-translate package. this server features:
- Authentication for API access
- Admin access to DBMS through API
- Redis DB Caching API responses to minimize API use
- PostgresSQL DBMS to store credentials
- Translate Text and Documents (PDF,SRT,TXT)
- Docker Compose for quick setup

## Quick Start

there is two ways to setup either through docker compose or from source.

### Docker Setup (Recommended)

This setup requires Docker and Git.

clone the repo:
```sh
git clone https://github.com/o0n1x/mass-translate-server.git
cd mass-translate-server
```

create a new env file:
```sh
nano docker/.env
```

paste the following into the env file:
```
DB_URL="postgres://postgresql:postgresql@postgresql:5432/masstranslate?sslmode=disable"
REDIS_URL="redis:6379"
SECRET_JWT=YOUR_SECRET_HERE
DEEPL_API=YOUR_API_KEY_HERE
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=password
```
replace `YOUR_API_KEY_HERE` with your deepl API key and Generate JWT secret with :
`openssl rand -base64 32`.

Note: you can change `ADMIN_EMAIL` and `ADMIN_PASSWORD` to your liking.

Note: If you changed DB credentials, update them in `docker-compose.yml` as well.

Start the server:
```sh
docker compose up -d
```

### Setup from source

This setup requires Git, Go 1.25+, Docker (for databases), and Goose (for database migration).


clone the repo:
```sh
git clone https://github.com/o0n1x/mass-translate-server.git
cd mass-translate-server
```

create a new env file:
```sh
nano .env
```
paste the following into the env file:
```
DB_URL="postgres://postgresql:postgresql@localhost:5432/masstranslate?sslmode=disable"
REDIS_URL="localhost:6379"
SECRET_JWT=YOUR_SECRET_HERE
DEEPL_API=YOUR_API_KEY_HERE
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=password
```

replace `YOUR_API_KEY_HERE` with your deepl API key and Generate JWT secret with :
`openssl rand -base64 32`.

setup PostgreSQL and Redis DB, I recommend using Docker compose:

Make sure you have docker installed.

create a docker compose file:
```sh
nano docker-compose.yml
```

paste the following compose:
```yaml
# services for mass translate server
services:
  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: postgresql
      POSTGRES_PASSWORD: postgresql
      POSTGRES_DB: masstranslate
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```
Note: If you changed DB credentials, update them here as well.

Run the docker compose file:
```sh
docker compose up -d
```

Install Goose (if not installed):
```sh
go install github.com/pressly/goose/v3/cmd/goose@latest
```
Run needed migrations:
```sh
goose -dir sql/schema postgres "postgres://postgresql:postgresql@localhost:5432/masstranslate?sslmode=disable" up
```

Build the project to an executable:
```sh
go build -o server .
```

run the server:
```sh
./server
```

## API Endpoints

[![OpenAPI](https://img.shields.io/badge/OpenAPI-3.0.4-green)](./docs/openapi.yaml)

Interactive docs: [Swagger Editor](https://editor.swagger.io/?url=https://raw.githubusercontent.com/o0n1x/mass-translate-server/refs/heads/main/docs/openapi.yaml)

All protected endpoints require:

```
Authorization: Bearer <token>
```


| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/api/health` | None | Health check |
| POST | `/api/deepl/translate` | User | Translate text and documents|
| POST | `/api/auth/login` | None | Login |
| POST | `/api/admin/users` | Admin | Create user |
| GET | `/api/admin/users` | Admin | List users |
| GET | `/api/admin/users/{id}` | Admin | Get user |
| DELETE | `/api/admin/users/{id}` | Admin | Delete user |
| PUT | `/api/admin/users/{id}` | Admin | Update user |


## Enviroment Variables


|Env Variable | Description |
|--------|----------|
| DB_URL | URL to the PostgreSQL DBMS |
|REDIS_URL | URL to Redis |
SECRET_JWT| a base64 32 digit long secret used to encrypt JWT tokens
DEEPL_API| Deepl API used for translation by the server 
ADMIN_EMAIL| default admin email for server access. Set to Nil to not setup admin account
ADMIN_PASSWORD | default admin password for server access

## Example API Requests

### Login
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "password"}'
```

### Translate Text
```bash
curl -X POST http://localhost:8080/api/deepl/translate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"text": ["Hello", "World"], "target_lang": "FR"}'
```

### Translate File

```bash
curl -X POST http://localhost:8080/api/deepl/translate \
  -H "Authorization: Bearer <token>" \
  -F "file=@example.txt" \
  -F "target_lang": "FR"
```
Set \<token\> to the token you got from login.

## Planned Features

- **Metrics**: gather metrics with prometheus and display it using graphana

- **Async document translation**: document translation become async due it taking a significant amount of time

- **Custom error package**: standardize errors to ease error handling in the future

