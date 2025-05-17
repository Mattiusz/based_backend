# based_backend

This project is the backend for an event application, designed to create and manage user events.

## Services Used

The application utilizes several services orchestrated via Docker Compose, as defined in [`.devcontainer/docker-compose.yml`](.devcontainer/docker-compose.yml ):

*   **Go**: The primary language for the backend application (cmd/server/main.go).
*   **PostgreSQL with PostGIS**: Used as the primary database for storing application data.
*   **Keycloak**: Provides authentication and authorization services.
*   **Redis**: Available for caching purposes.
*   **Supabase Studio & pgMeta**: Included in the `full` profile for database management and introspection.
*   **gRPC**: Used for communication between services. The protobuf definitions can be found in the [`proto`](proto ) directory (e.g., [`proto/event.proto`](proto/event.proto ), proto/user.proto).

## Getting Started

The [`Makefile`](Makefile ) provides several commands to build, run, and manage the application and its dependencies.

Key commands include:

*   `make build`: Builds the Go application binary.
*   `make run`: Builds and runs the Go application locally.
*   `make protoc`: Generates Go code from the protobuf definitions.
*   [`make sqlc`](cmd/server/main.go ): Generates Go code from SQL queries.
*   `make migrate-up`: Applies database migrations.
*   `make docker-build`: Builds the application's Docker image.
*   `make docker-light`: Starts a lightweight Docker environment with essential services.
*   `make docker-full`: Starts a full Docker environment including development tools like Supabase Studio.

Please refer to the [`Makefile`](Makefile ) for a complete list of commands and their descriptions.