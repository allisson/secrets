# ➕ Adding New Domains

This guide walks you through adding a new domain to the project. We'll use a `product` domain as an example.

## 🏗️ Domain Structure

Each domain follows this structure:

```
internal/product/
├── domain/
│   └── product.go              # Entity + domain errors
├── usecase/
│   └── product_usecase.go      # UseCase interface + implementation
├── repository/
│   ├── mysql_product_repository.go
│   └── postgresql_product_repository.go
└── http/
    ├── dto/
    │   ├── request.go           # Request DTOs with validation
    │   ├── response.go          # Response DTOs with JSON tags
    │   └── mapper.go            # DTO ↔ domain conversions
    └── product_handler.go       # HTTP handlers
```

## 📝 Step-by-Step Guide

### Step 1: Create Domain Entity

Create `internal/product/domain/product.go`:

```go
package domain

import (
    "time"
    
    "github.com/google/uuid"
    
    apperrors "github.com/yourproject/internal/errors"
)

// Product represents a product in the system
type Product struct {
    ID          uuid.UUID
    Name        string
    Description string
    Price       float64
    Stock       int
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Domain-specific errors
var (
    ErrProductNotFound    = apperrors.Wrap(apperrors.ErrNotFound, "product not found")
    ErrInsufficientStock  = apperrors.Wrap(apperrors.ErrConflict, "insufficient stock")
    ErrInvalidPrice       = apperrors.Wrap(apperrors.ErrInvalidInput, "invalid price")
    ErrInvalidQuantity    = apperrors.Wrap(apperrors.ErrInvalidInput, "quantity must be positive")
)
```

**Key Points**:
- ✅ No JSON tags (domain models are pure)
- ✅ Use `uuid.UUID` for ID
- ✅ Wrap standard errors with domain-specific messages

### Step 2: Create Database Migrations

Create migration files for both databases:

**PostgreSQL** (`migrations/postgresql/000003_create_products_table.up.sql`):

```sql
CREATE TABLE products (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_products_name ON products(name);
```

**PostgreSQL Down** (`migrations/postgresql/000003_create_products_table.down.sql`):

```sql
DROP INDEX IF EXISTS idx_products_name;
DROP TABLE IF EXISTS products;
```

**MySQL** (`migrations/mysql/000003_create_products_table.up.sql`):

```sql
CREATE TABLE products (
    id BINARY(16) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_products_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**MySQL Down** (`migrations/mysql/000003_create_products_table.down.sql`):

```sql
DROP TABLE IF EXISTS products;
```

Run migrations:
```bash
make run-migrate
```

### Step 3: Create Repository Interface & Implementations

**Define Repository Interface** in `internal/product/usecase/product_usecase.go`:

```go
package usecase

import (
    "context"
    
    "github.com/google/uuid"
    
    "github.com/yourproject/internal/product/domain"
)

// ProductRepository defines the interface for product data access
type ProductRepository interface {
    Create(ctx context.Context, product *domain.Product) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
    Update(ctx context.Context, product *domain.Product) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, limit, offset int) ([]*domain.Product, error)
}
```

**PostgreSQL Implementation** (`internal/product/repository/postgresql_product_repository.go`):

```go
package repository

import (
    "context"
    "database/sql"
    "errors"
    "strings"
    
    "github.com/google/uuid"
    
    apperrors "github.com/yourproject/internal/errors"
    "github.com/yourproject/internal/database"
    "github.com/yourproject/internal/product/domain"
)

type PostgreSQLProductRepository struct {
    db *sql.DB
}

func NewPostgreSQLProductRepository(db *sql.DB) *PostgreSQLProductRepository {
    return &PostgreSQLProductRepository{db: db}
}

func (r *PostgreSQLProductRepository) Create(ctx context.Context, product *domain.Product) error {
    querier := database.GetTx(ctx, r.db)
    
    query := `INSERT INTO products (id, name, description, price, stock, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`
    
    _, err := querier.ExecContext(ctx, query,
        product.ID,
        product.Name,
        product.Description,
        product.Price,
        product.Stock,
    )
    if err != nil {
        if isPostgreSQLUniqueViolation(err) {
            return apperrors.Wrap(apperrors.ErrConflict, "product with this name already exists")
        }
        return apperrors.Wrap(err, "failed to create product")
    }
    return nil
}

func (r *PostgreSQLProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
    querier := database.GetTx(ctx, r.db)
    
    query := `SELECT id, name, description, price, stock, created_at, updated_at 
              FROM products WHERE id = $1`
    
    var product domain.Product
    err := querier.QueryRowContext(ctx, query, id).Scan(
        &product.ID,
        &product.Name,
        &product.Description,
        &product.Price,
        &product.Stock,
        &product.CreatedAt,
        &product.UpdatedAt,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, domain.ErrProductNotFound
        }
        return nil, apperrors.Wrap(err, "failed to get product by id")
    }
    return &product, nil
}

func isPostgreSQLUniqueViolation(err error) bool {
    return strings.Contains(err.Error(), "duplicate key") ||
           strings.Contains(err.Error(), "unique constraint")
}
```

**MySQL Implementation** (`internal/product/repository/mysql_product_repository.go`):

```go
package repository

import (
    "context"
    "database/sql"
    "errors"
    "strings"
    
    "github.com/google/uuid"
    
    apperrors "github.com/yourproject/internal/errors"
    "github.com/yourproject/internal/database"
    "github.com/yourproject/internal/product/domain"
)

type MySQLProductRepository struct {
    db *sql.DB
}

func NewMySQLProductRepository(db *sql.DB) *MySQLProductRepository {
    return &MySQLProductRepository{db: db}
}

func (r *MySQLProductRepository) Create(ctx context.Context, product *domain.Product) error {
    querier := database.GetTx(ctx, r.db)
    
    query := `INSERT INTO products (id, name, description, price, stock, created_at, updated_at) 
              VALUES (?, ?, ?, ?, ?, NOW(), NOW())`
    
    // Convert UUID to bytes for MySQL BINARY(16)
    uuidBytes, err := product.ID.MarshalBinary()
    if err != nil {
        return apperrors.Wrap(err, "failed to marshal UUID")
    }
    
    _, err = querier.ExecContext(ctx, query,
        uuidBytes,
        product.Name,
        product.Description,
        product.Price,
        product.Stock,
    )
    if err != nil {
        if isMySQLUniqueViolation(err) {
            return apperrors.Wrap(apperrors.ErrConflict, "product with this name already exists")
        }
        return apperrors.Wrap(err, "failed to create product")
    }
    return nil
}

func isMySQLUniqueViolation(err error) bool {
    return strings.Contains(err.Error(), "Error 1062") ||
           strings.Contains(err.Error(), "Duplicate entry")
}
```

### Step 4: Create Use Case

Create `internal/product/usecase/product_usecase.go`:

```go
package usecase

import (
    "context"
    
    "github.com/google/uuid"
    validation "github.com/jellydator/validation"
    
    apperrors "github.com/yourproject/internal/errors"
    "github.com/yourproject/internal/database"
    "github.com/yourproject/internal/product/domain"
)

// UseCase defines the interface for product business logic
type UseCase interface {
    CreateProduct(ctx context.Context, input CreateProductInput) (*domain.Product, error)
    GetProduct(ctx context.Context, id uuid.UUID) (*domain.Product, error)
    UpdateProduct(ctx context.Context, id uuid.UUID, input UpdateProductInput) (*domain.Product, error)
    DeleteProduct(ctx context.Context, id uuid.UUID) error
    ListProducts(ctx context.Context, limit, offset int) ([]*domain.Product, error)
}

type ProductUseCase struct {
    txManager   database.TxManager
    productRepo ProductRepository
}

func NewProductUseCase(txManager database.TxManager, productRepo ProductRepository) *ProductUseCase {
    return &ProductUseCase{
        txManager:   txManager,
        productRepo: productRepo,
    }
}

// CreateProductInput represents the input for creating a product
type CreateProductInput struct {
    Name        string
    Description string
    Price       float64
    Stock       int
}

func (i CreateProductInput) Validate() error {
    return validation.ValidateStruct(&i,
        validation.Field(&i.Name, validation.Required, validation.Length(1, 255)),
        validation.Field(&i.Price, validation.Required, validation.Min(0.0)),
        validation.Field(&i.Stock, validation.Required, validation.Min(0)),
    )
}

func (uc *ProductUseCase) CreateProduct(ctx context.Context, input CreateProductInput) (*domain.Product, error) {
    if err := input.Validate(); err != nil {
        return nil, apperrors.Wrap(apperrors.ErrInvalidInput, err.Error())
    }
    
    if input.Price <= 0 {
        return nil, domain.ErrInvalidPrice
    }
    
    if input.Stock < 0 {
        return nil, domain.ErrInvalidQuantity
    }
    
    product := &domain.Product{
        ID:          uuid.Must(uuid.NewV7()),
        Name:        input.Name,
        Description: input.Description,
        Price:       input.Price,
        Stock:       input.Stock,
    }
    
    if err := uc.productRepo.Create(ctx, product); err != nil {
        return nil, err
    }
    
    return product, nil
}

func (uc *ProductUseCase) GetProduct(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
    return uc.productRepo.GetByID(ctx, id)
}
```

### Step 5: Create DTOs

**Request DTO** (`internal/product/http/dto/request.go`):

```go
package dto

import (
    validation "github.com/jellydator/validation"
    
    appValidation "github.com/yourproject/internal/validation"
)

type CreateProductRequest struct {
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Price       float64 `json:"price"`
    Stock       int     `json:"stock"`
}

func (r *CreateProductRequest) Validate() error {
    err := validation.ValidateStruct(r,
        validation.Field(&r.Name,
            validation.Required.Error("name is required"),
            appValidation.NotBlank,
            validation.Length(1, 255),
        ),
        validation.Field(&r.Price,
            validation.Required.Error("price is required"),
            validation.Min(0.01).Error("price must be greater than 0"),
        ),
        validation.Field(&r.Stock,
            validation.Required.Error("stock is required"),
            validation.Min(0).Error("stock cannot be negative"),
        ),
    )
    return appValidation.WrapValidationError(err)
}
```

**Response DTO** (`internal/product/http/dto/response.go`):

```go
package dto

import (
    "time"
    
    "github.com/google/uuid"
)

type ProductResponse struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Price       float64   `json:"price"`
    Stock       int       `json:"stock"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

**Mapper** (`internal/product/http/dto/mapper.go`):

```go
package dto

import (
    "github.com/yourproject/internal/product/domain"
    "github.com/yourproject/internal/product/usecase"
)

func ToCreateProductInput(req CreateProductRequest) usecase.CreateProductInput {
    return usecase.CreateProductInput{
        Name:        req.Name,
        Description: req.Description,
        Price:       req.Price,
        Stock:       req.Stock,
    }
}

func ToProductResponse(product *domain.Product) ProductResponse {
    return ProductResponse{
        ID:          product.ID,
        Name:        product.Name,
        Description: product.Description,
        Price:       product.Price,
        Stock:       product.Stock,
        CreatedAt:   product.CreatedAt,
        UpdatedAt:   product.UpdatedAt,
    }
}

func ToProductListResponse(products []*domain.Product) []ProductResponse {
    responses := make([]ProductResponse, len(products))
    for i, product := range products {
        responses[i] = ToProductResponse(product)
    }
    return responses
}
```

### Step 6: Create HTTP Handler

Create `internal/product/http/product_handler.go`:

```go
package http

import (
    "encoding/json"
    "log/slog"
    "net/http"
    
    "github.com/google/uuid"
    
    "github.com/yourproject/internal/httputil"
    "github.com/yourproject/internal/product/http/dto"
    "github.com/yourproject/internal/product/usecase"
)

type ProductHandler struct {
    productUseCase usecase.UseCase
    logger         *slog.Logger
}

func NewProductHandler(productUseCase usecase.UseCase, logger *slog.Logger) *ProductHandler {
    return &ProductHandler{
        productUseCase: productUseCase,
        logger:         logger,
    }
}

func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
    var req dto.CreateProductRequest
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        httputil.HandleValidationError(w, err, h.logger)
        return
    }
    
    if err := req.Validate(); err != nil {
        httputil.HandleError(w, err, h.logger)
        return
    }
    
    input := dto.ToCreateProductInput(req)
    product, err := h.productUseCase.CreateProduct(r.Context(), input)
    if err != nil {
        httputil.HandleError(w, err, h.logger)
        return
    }
    
    response := dto.ToProductResponse(product)
    httputil.MakeJSONResponse(w, http.StatusCreated, response)
}

func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
    // Parse product ID from URL
    idStr := r.PathValue("id")
    id, err := uuid.Parse(idStr)
    if err != nil {
        httputil.HandleValidationError(w, err, h.logger)
        return
    }
    
    product, err := h.productUseCase.GetProduct(r.Context(), id)
    if err != nil {
        httputil.HandleError(w, err, h.logger)
        return
    }
    
    response := dto.ToProductResponse(product)
    httputil.MakeJSONResponse(w, http.StatusOK, response)
}
```

### Step 7: Register in DI Container

Update `internal/app/di.go`:

```go
type Container struct {
    // ... existing fields
    productRepo        productUsecase.ProductRepository
    productUseCase     productUsecase.UseCase  // Interface, not concrete type!
    productRepoInit    sync.Once
    productUseCaseInit sync.Once
}

func (c *Container) ProductRepository() (productUsecase.ProductRepository, error) {
    var err error
    c.productRepoInit.Do(func() {
        c.productRepo, err = c.initProductRepository()
        if err != nil {
            c.initErrors["productRepo"] = err
        }
    })
    if err != nil {
        return nil, err
    }
    if existingErr, ok := c.initErrors["productRepo"]; ok {
        return nil, existingErr
    }
    return c.productRepo, nil
}

func (c *Container) ProductUseCase() (productUsecase.UseCase, error) {
    var err error
    c.productUseCaseInit.Do(func() {
        c.productUseCase, err = c.initProductUseCase()
        if err != nil {
            c.initErrors["productUseCase"] = err
        }
    })
    if err != nil {
        return nil, err
    }
    if existingErr, ok := c.initErrors["productUseCase"]; ok {
        return nil, existingErr
    }
    return c.productUseCase, nil
}

func (c *Container) initProductRepository() (productUsecase.ProductRepository, error) {
    db, err := c.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get database: %w", err)
    }
    
    switch c.config.DBDriver {
    case "mysql":
        return productRepository.NewMySQLProductRepository(db), nil
    case "postgres":
        return productRepository.NewPostgreSQLProductRepository(db), nil
    default:
        return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
    }
}

func (c *Container) initProductUseCase() (productUsecase.UseCase, error) {
    txManager, err := c.TxManager()
    if err != nil {
        return nil, fmt.Errorf("failed to get tx manager: %w", err)
    }
    
    productRepo, err := c.ProductRepository()
    if err != nil {
        return nil, fmt.Errorf("failed to get product repository: %w", err)
    }
    
    return productUsecase.NewProductUseCase(txManager, productRepo), nil
}
```

### Step 8: Wire HTTP Routes

Update `internal/http/server.go`:

```go
func (s *Server) setupRoutes() {
    mux := http.NewServeMux()
    
    // Health checks
    mux.HandleFunc("/health", s.handleHealth)
    mux.HandleFunc("/ready", s.handleReady)
    
    // User routes
    userHandler := userHttp.NewUserHandler(s.container.UserUseCase(), s.logger)
    mux.HandleFunc("POST /api/users", userHandler.RegisterUser)
    
    // Product routes
    productHandler := productHttp.NewProductHandler(s.container.ProductUseCase(), s.logger)
    mux.HandleFunc("POST /api/products", productHandler.CreateProduct)
    mux.HandleFunc("GET /api/products/{id}", productHandler.GetProduct)
    
    s.handler = s.middleware(mux)
}
```

### Step 9: Write Tests

Create tests for your new domain following the [Testing Guide](testing.md).

### Step 10: Test Your New Domain

```bash
# Run migrations
make run-migrate

# Start server
make run-server

# Create a product
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Laptop",
    "description": "High-performance laptop",
    "price": 999.99,
    "stock": 10
  }'

# Get a product
curl http://localhost:8080/api/products/{id}
```

## ✅ Checklist

Use this checklist when adding a new domain:

- [ ] Create domain entity with business logic
- [ ] Define domain-specific errors
- [ ] Create database migrations (PostgreSQL and MySQL)
- [ ] Run migrations
- [ ] Define repository interface in use case package
- [ ] Implement PostgreSQL repository
- [ ] Implement MySQL repository
- [ ] Create use case interface
- [ ] Implement use case with business logic
- [ ] Create request DTOs with validation
- [ ] Create response DTOs with JSON tags
- [ ] Create mapper functions
- [ ] Create HTTP handlers
- [ ] Register repositories in DI container
- [ ] Register use cases in DI container
- [ ] Wire HTTP routes in server
- [ ] Write repository tests (PostgreSQL and MySQL)
- [ ] Write use case tests
- [ ] Write HTTP handler tests
- [ ] Run linter and fix issues
- [ ] Test endpoints manually
- [ ] Update documentation if needed

## 🎯 Best Practices

- ✅ **Follow existing patterns** - Look at the `user` domain for reference
- ✅ **Keep domain models pure** - No JSON tags, no framework dependencies
- ✅ **Define clear interfaces** - Use case and repository interfaces
- ✅ **Wrap standard errors** - Create domain-specific errors
- ✅ **Validate input** - At both DTO and use case levels
- ✅ **Use transactions** - When multiple operations must be atomic
- ✅ **Write tests** - Integration tests with real databases
- ✅ **Document code** - Add comments for complex logic
- ✅ **Run linter** - Before committing changes

## 📚 Related Documentation

- [Architecture](architecture.md) - Understand the architectural patterns
- [Error Handling](error-handling.md) - Learn the error handling system
- [Development](development.md) - Development workflow and guidelines
- [Testing](testing.md) - Writing effective tests
