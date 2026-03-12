```
Backend/
├── agent_db.txt
├── docker-compose.yaml
├── Dockerfile
├── go.mod
├── README.md
├── build/
│   ├── ci-cd.yml
│   ├── docker-compose.yml
│   ├── Dockerfile
│   └── Makefile
├── cmd/
│   ├── api/
│   │   └── main.go
│   ├── grpc/
│   │   └── main.go
│   └── worker/
│       └── main.go
├── configs/
│   ├── config.yaml
│   ├── local.env
│   └── prod.env
├── docs/
│   └── product-discovery-engine-architecture.md
├── internal/
│   ├── app/
│   │   ├── router.go
│   │   └── server.go
│   ├── config/
│   │   └── config.go
│   ├── db/
│   │   ├── postgres.go
│   │   ├── redis.go
│   │   └── migrations/
│   │       ├── 000_auth_init.sql
│   │       ├── 001_svc_init.sql
│   │       ├── 002_category_closure_triggers.sql
│   │       └── …
│   ├── domain/
│   │   ├── admin/
│   │   ├── attribute/
│   │   ├── auth/
│   │   ├── brand/
│   │   ├── category/
│   │   ├── discount/
│   │   ├── discovery/
│   │   ├── inventory/
│   │   ├── notification/
│   │   ├── product/
│   │   ├── tag/
│   │   ├── user/
│   │   ├── variant/
│   │   └── websocket/
│   ├── handlers/
│   │   ├── auth/
│   │   ├── catalog/
│   │   ├── discovery/
│   │   ├── notification/
│   │   ├── user/
│   │   └── websocket/
│   ├── middleware/
│   │   ├── auth_middleware.go
│   │   ├── cors_middleware.go
│   │   ├── helpers.go
│   │   ├── logging_middleware.go
│   │   ├── recovery_middleware.go
│   │   └── …
│   ├── pkg/
│   │   ├── crypto/
│   │   ├── errors/
│   │   ├── jwt/
│   │   ├── logger/
│   │   ├── response/
│   │   ├── session/
│   │   └── utils/
│   ├── proto/
│   │   ├── driver.proto
│   │   ├── trip.proto
│   │   └── wallet.proto
│   ├── repository/
│   │   ├── postgres/
│   │   └── redis/
│   ├── service/
│   │   ├── auth/
│   │   ├── catalog/
│   │   ├── discovery/
│   │   ├── email/
│   │   ├── notification/
│   │   ├── user/
│   │   └── worker/
│   ├── tests/
│   │   └── image.go
│   └── websocket/
│       ├── client.go
│       ├── errors.go
│       ├── handler.go
│       ├── hub.go
│       ├── utils.go
│       └── handler/
├── postman/
│   ├── authy.postman_collection.json
│   └── zentora.postman_collection.json
├── scripts/
│   ├── migrate.sh
│   ├── run.sh
│   └── seed.sh
└── static/
    └── assets/
        └── pages/
            └── password_reset.html

```