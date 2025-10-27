# Type-Safe Go API Framework

A modern, type-safe REST API framework for Go that leverages generics to provide compile-time guarantees, automatic request validation, and built-in OpenAPI documentation generation.

## Features

- **Type Safety**: Full compile-time type checking using Go generics
- **Automatic Validation**: Built-in request validation using struct tags
- **Request Parsing**: Automatic parsing of headers, route parameters, query strings, and JSON bodies
- **Query Arrays**: Native support for multiple values in query parameters
- **File Uploads**: Type-safe file upload handling with multipart form data
- **OpenAPI Generation**: Auto-generated OpenAPI 3.0 specs and Swagger UI with file upload support
- **Route Grouping**: Organize routes with path prefixes and shared middleware
- **Middleware Support**: Standard HTTP middleware at framework, group, and endpoint levels
- **Zero Boilerplate**: Write handlers as simple functions that return typed responses

## Installation

```bash
go get github.com/example/api-framework/framework
go get github.com/example/api-framework/framework/openapi
```

## Quick Start

```go
package main

import (
    "fmt"
    "net/http"
    "log"

    "github.com/example/api-framework/framework"
    "github.com/example/api-framework/framework/openapi"
)

// Define your request and response types
type CreateUserRequest struct {
    Body struct {
        Name  string `json:"name" validate:"required,min=3,max=50"`
        Email string `json:"email" validate:"required,email"`
    } `body:"" validate:"required"`
}

type CreateUserResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Write a type-safe handler
func CreateUser(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
    // Request is already validated and parsed
    // ctx contains request context for cancellation, timeouts, etc.
    return CreateUserResponse{
        ID:    "123",
        Name:  req.Body.Name,
        Email: req.Body.Email,
    }, nil
}

func main() {
    app := framework.New()

    // Register endpoint with fluent API
    framework.POST("/users", CreateUser).
        Summary("Create User").
        Description("Creates a new user").
        Tags("Users").
        Register(app)

    // Register OpenAPI docs
    openapi := openapi.NewOpenApi(app)
    openapi.RegisterOpenAPIDocs(
        "My API",
        "A type-safe REST API",
        "1.0.0",
        "/openapi.json",
        "/docs",
    )

    log.Fatal(http.ListenAndServe(":8080", app))
}
```

## Request Binding

The framework automatically binds request data to your request structs using tags:

### Route Parameters

```go
type GetUserRequest struct {
    UserID string `route:"id" validate:"required" doc:"User ID"`
}

// Matches: GET /users/{id}
framework.GET("/users/{id}", GetUser).Register(app)
```

### Query Parameters

```go
type ListUsersRequest struct {
    Page     int    `query:"page" validate:"min=1"`
    PageSize int    `query:"page_size" validate:"min=1,max=100"`
    SortBy   string `query:"sort_by" validate:"oneof=name email created_at"`
}

// Matches: GET /users?page=1&page_size=10&sort_by=name
framework.GET("/users", ListUsers).Register(app)
```

### Query Arrays

Use slice types to accept multiple values for the same query parameter:

```go
type FilterUsersRequest struct {
    Tags     []string `query:"tags" doc:"Filter by tags"`
    Statuses []string `query:"status" validate:"omitempty,dive,oneof=active inactive"`
    IDs      []int    `query:"id" doc:"Filter by multiple IDs"`
}

// Matches: GET /users?tags=admin&tags=premium&status=active&id=1&id=2&id=3
framework.GET("/users/filter", FilterUsers).Register(app)

// In your handler:
func FilterUsers(ctx context.Context, req FilterUsersRequest) (Response, error) {
    // req.Tags = ["admin", "premium"]
    // req.Statuses = ["active"]
    // req.IDs = [1, 2, 3]

    // Use the arrays to filter results
    for _, tag := range req.Tags {
        // Filter logic
    }
}
```

**Supported Array Types:**
- `[]string` - String arrays
- `[]int`, `[]int32`, `[]int64` - Integer arrays
- `[]uint`, `[]uint32`, `[]uint64` - Unsigned integer arrays
- `[]bool` - Boolean arrays
- `[]float32`, `[]float64` - Float arrays

### Headers

```go
type AuthenticatedRequest struct {
    APIKey      string `header:"X-API-Key" validate:"required"`
    ContentType string `header:"Content-Type" validate:"required,eq=application/json"`
}
```

### Request Body

```go
type UpdateUserRequest struct {
    UserID string `route:"id" validate:"required"`

    Body struct {
        Name  string `json:"name,omitempty" validate:"omitempty,min=3,max=50"`
        Email string `json:"email,omitempty" validate:"omitempty,email"`
        Age   int    `json:"age,omitempty" validate:"omitempty,min=18,max=120"`
    } `body:"" validate:"required"`
}
```

### File Uploads

Handle file uploads with type-safe multipart form data:

```go
import "github.com/example/api-framework/framework"

type UploadAvatarRequest struct {
    UserID string               `route:"id" validate:"required"`
    APIKey string               `header:"X-API-Key" validate:"required"`
    Avatar framework.FileField `form:"avatar" validate:"required" doc:"Avatar image file"`
}

type UploadAvatarResponse struct {
    Message  string `json:"message"`
    Filename string `json:"filename"`
    Size     int64  `json:"size"`
}

func UploadAvatar(ctx context.Context, req UploadAvatarRequest) (UploadAvatarResponse, error) {
    // IMPORTANT: Always close the file when done!
    defer req.Avatar.Content.Close()

    // Access file properties
    filename := req.Avatar.Filename           // Original filename
    size := req.Avatar.Size                   // File size in bytes
    header := req.Avatar.Header               // Full multipart header
    content := req.Avatar.Content             // io.ReadCloser to read file

    // Read file content
    data, err := io.ReadAll(content)
    if err != nil {
        return UploadAvatarResponse{}, fmt.Errorf("failed to read file: %w", err)
    }

    // Validate file type
    if !strings.HasPrefix(http.DetectContentType(data), "image/") {
        return UploadAvatarResponse{}, fmt.Errorf("file must be an image")
    }

    // Save to storage (S3, local disk, etc.)
    // ... your storage logic ...

    return UploadAvatarResponse{
        Message:  "Upload successful",
        Filename: filename,
        Size:     size,
    }, nil
}

// Register the endpoint
framework.POST("/users/{id}/avatar", UploadAvatar).
    Summary("Upload User Avatar").
    Description("Upload an avatar image for a user").
    Tags("Users").
    Register(app)
```

**FileField Properties:**
- `Filename` (string) - Original filename from the client
- `Size` (int64) - File size in bytes
- `Header` (*multipart.FileHeader) - Full multipart header with metadata
- `Content` (io.ReadCloser) - Stream to read file content

**Usage with cURL:**
```bash
curl -X POST http://localhost:8080/users/123/avatar \
  -H "X-API-Key: secret-key" \
  -F "avatar=@/path/to/image.jpg"
```

**Important Notes:**
- Always `defer req.Avatar.Content.Close()` to prevent memory leaks
- File uploads use `multipart/form-data` content type
- Default max memory is 32MB (configurable per upload)
- OpenAPI spec automatically shows file picker in Swagger UI

### Combining Multiple Sources

You can combine route parameters, headers, query parameters, form data, and request body in a single request:

```go
type CompleteRequest struct {
    // Route parameter
    UserID string `route:"id" validate:"required"`

    // Headers
    APIKey string `header:"X-API-Key" validate:"required"`

    // Query parameters
    IncludeDetails bool     `query:"include_details"`
    Tags           []string `query:"tags"`

    // File upload
    Avatar framework.FileField `form:"avatar" validate:"required"`

    // Note: You cannot mix `form` and `body` tags in the same request
    // Use either form data OR JSON body, not both
}
```

**Important:** The `form` tag (file uploads) and `body` tag (JSON) are mutually exclusive. A request can either be:
- `application/json` with a `body` field
- `multipart/form-data` with `form` fields

You cannot have both in the same request.

## Validation

The framework uses [go-playground/validator](https://github.com/go-playground/validator) for validation. Common validation tags:

```go
type ValidationExample struct {
    // Required fields
    Name string `json:"name" validate:"required"`

    // String length
    Username string `json:"username" validate:"min=3,max=20"`

    // Email validation
    Email string `json:"email" validate:"required,email"`

    // Numeric ranges
    Age int `json:"age" validate:"min=18,max=120"`

    // Enum values
    Status string `json:"status" validate:"oneof=active inactive pending"`

    // Optional fields with validation when present
    Phone string `json:"phone,omitempty" validate:"omitempty,min=10"`
}
```

When validation fails, the framework automatically returns a structured error:

```json
{
  "error": "validation failed",
  "fields": [
    {
      "field": "name",
      "source_type": "body",
      "errors": ["failed validation: required"]
    },
    {
      "field": "age",
      "source_type": "body",
      "errors": ["failed validation: min"]
    }
  ]
}
```

## HTTP Methods

The framework supports all standard HTTP methods with a fluent API:

```go
// GET request
framework.GET("/users", ListUsers).
    Summary("List Users").
    Tags("Users").
    Register(app)

// POST request
framework.POST("/users", CreateUser).
    Summary("Create User").
    Tags("Users").
    Register(app)

// PUT request
framework.PUT("/users/{id}", UpdateUser).
    Summary("Update User").
    Tags("Users").
    Register(app)

// PATCH request
framework.PATCH("/users/{id}", PatchUser).
    Summary("Patch User").
    Tags("Users").
    Register(app)

// DELETE request
framework.DELETE("/users/{id}", DeleteUser).
    Summary("Delete User").
    Tags("Users").
    Register(app)
```

## Route Groups

Organize your API with route groups and shared middleware:

```go
app := framework.New()

// Create /api/v1 group
api := app.Group("/api/v1").Use(LoggingMiddleware)

// Create /api/v1/users group
users := api.Group("/users").Use(AuthMiddleware)

// All routes registered on 'users' will be prefixed with /api/v1/users
// and have both logging and auth middleware applied

framework.GET("", ListUsers).Register(users)          // GET /api/v1/users
framework.POST("", CreateUser).Register(users)        // POST /api/v1/users
framework.GET("/{id}", GetUser).Register(users)       // GET /api/v1/users/{id}
framework.PUT("/{id}", UpdateUser).Register(users)    // PUT /api/v1/users/{id}
framework.DELETE("/{id}", DeleteUser).Register(users) // DELETE /api/v1/users/{id}
```

### Nested Groups

Groups can be nested to create hierarchical route structures:

```go
api := app.Group("/api")
v1 := api.Group("/v1")
users := v1.Group("/users")
admin := users.Group("/admin")

// Routes in 'admin' group will have prefix /api/v1/users/admin
framework.GET("", ListAdminUsers).Register(admin) // GET /api/v1/users/admin
```

## Middleware

### Framework-Level Middleware

Applied to all routes:

```go
app := framework.New()
// Not directly supported - use groups or endpoint-level middleware
```

### Group-Level Middleware

Applied to all routes in a group:

```go
api := app.Group("/api").Use(LoggingMiddleware, CORSMiddleware)
users := api.Group("/users").Use(AuthMiddleware)

// Routes in 'users' have all three middleware: Logging, CORS, Auth
```

### Endpoint-Level Middleware

Applied to specific endpoints:

```go
framework.GET("/admin/users", ListUsers).
    Use(AdminMiddleware, RateLimitMiddleware).
    Register(app)
```

### Writing Middleware

Middleware follows the standard Go HTTP middleware pattern:

```go
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Printf("[%s] %s %s\n", r.Method, r.URL.Path, r.RemoteAddr)
        next.ServeHTTP(w, r)
    })
}

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        apiKey := r.Header.Get("X-API-Key")
        if apiKey == "" {
            http.Error(w, `{"error":"Missing API key"}`, http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### Middleware Execution Order

Middleware is executed in the order added, creating nested layers:

```go
// Group middleware first, then endpoint middleware
group.Use(A, B)  // A wraps B
endpoint.Use(C)  // Execution: A -> B -> C -> handler
```

## OpenAPI Documentation

Automatically generate OpenAPI 3.0 specifications and interactive documentation:

```go
openapi := openapi.NewOpenApi(app)
err := openapi.RegisterOpenAPIDocs(
    "User API",                              // API title
    "A fully type-safe REST API",            // Description
    "1.0.0",                                 // Version
    "/openapi.json",                         // OpenAPI spec endpoint
    "/docs",                                 // Swagger UI endpoint
)
```

Access your documentation at:
- **Swagger UI**: `http://localhost:8080/docs`
- **OpenAPI Spec**: `http://localhost:8080/openapi.json`

### Documentation Tags

Add documentation to your endpoints and fields:

```go
// Endpoint documentation
framework.POST("/users", CreateUser).
    Summary("Create a new user").
    Description("Creates a user with the provided information. Requires authentication.").
    Tags("Users", "Management").
    Register(app)

// Field documentation
type CreateUserRequest struct {
    APIKey string `header:"X-API-Key" validate:"required" doc:"API authentication key"`

    Body struct {
        Name  string `json:"name" validate:"required,min=3" doc:"User's full name"`
        Email string `json:"email" validate:"required,email" doc:"User's email address"`
        Age   int    `json:"age" validate:"required,min=18" doc:"User's age (must be 18+)"`
    } `body:"" doc:"User data to create"`
}
```

## Error Handling

### Handler Errors

Return errors from your handlers for automatic error responses:

```go
func GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    user, exists := users[req.UserID]
    if !exists {
        return GetUserResponse{}, fmt.Errorf("user not found")
    }
    return GetUserResponse{User: user}, nil
}
```

Error response (500 Internal Server Error):
```json
{
  "error": "user not found"
}
```

### Custom Error Responses

For more control, you can use custom status codes via middleware or by implementing the `Responder` interface.

### Empty Responses

Return empty structs for 204 No Content responses:

```go
type DeleteUserResponse struct{}

func DeleteUser(ctx context.Context, req DeleteUserRequest) (DeleteUserResponse, error) {
    delete(users, req.UserID)
    return DeleteUserResponse{}, nil
}
```

## Complete Example

Here's a complete example showing all features:

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/example/api-framework/framework"
    "github.com/example/api-framework/framework/openapi"
)

// Models
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

// Requests
type CreateUserRequest struct {
    APIKey string `header:"X-API-Key" validate:"required" doc:"API key"`

    Body struct {
        Name  string `json:"name" validate:"required,min=3,max=50" doc:"User's name"`
        Email string `json:"email" validate:"required,email" doc:"User's email"`
        Age   int    `json:"age" validate:"required,min=18,max=120" doc:"User's age"`
    } `body:"" validate:"required"`
}

type GetUserRequest struct {
    UserID string `route:"id" validate:"required" doc:"User ID"`
    APIKey string `header:"X-API-Key" validate:"required"`
}

type ListUsersRequest struct {
    APIKey   string `header:"X-API-Key" validate:"required"`
    Page     int    `query:"page" validate:"omitempty,min=1" doc:"Page number"`
    PageSize int    `query:"page_size" validate:"omitempty,min=1,max=100" doc:"Items per page"`
}

// Responses
type CreateUserResponse struct {
    User User `json:"user"`
}

type GetUserResponse struct {
    User User `json:"user"`
}

type ListUsersResponse struct {
    Users      []User                 `json:"users"`
    Pagination map[string]interface{} `json:"pagination"`
}

// Storage
var users = make(map[string]User)
var userCounter = 0

// Handlers
func CreateUser(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
    userCounter++
    user := User{
        ID:    fmt.Sprintf("user-%d", userCounter),
        Name:  req.Body.Name,
        Email: req.Body.Email,
        Age:   req.Body.Age,
    }
    users[user.ID] = user
    return CreateUserResponse{User: user}, nil
}

func GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    user, exists := users[req.UserID]
    if !exists {
        return GetUserResponse{}, fmt.Errorf("user not found")
    }
    return GetUserResponse{User: user}, nil
}

func ListUsers(ctx context.Context, req ListUsersRequest) (ListUsersResponse, error) {
    page := req.Page
    if page == 0 {
        page = 1
    }

    pageSize := req.PageSize
    if pageSize == 0 {
        pageSize = 10
    }

    userList := make([]User, 0, len(users))
    for _, user := range users {
        userList = append(userList, user)
    }

    return ListUsersResponse{
        Users: userList,
        Pagination: map[string]interface{}{
            "page":        page,
            "page_size":   pageSize,
            "total":       len(userList),
            "total_pages": (len(userList) + pageSize - 1) / pageSize,
        },
    }, nil
}

// Middleware
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Printf("[%s] %s\n", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        apiKey := r.Header.Get("X-API-Key")
        if apiKey != "secret-key" {
            http.Error(w, `{"error":"Invalid API key"}`, http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func main() {
    app := framework.New()

    // Create route groups
    api := app.Group("/api/v1").Use(LoggingMiddleware)
    usersGroup := api.Group("/users").Use(AuthMiddleware)

    // Register endpoints
    framework.POST("", CreateUser).
        Summary("Create User").
        Description("Creates a new user").
        Tags("Users").
        Register(usersGroup)

    framework.GET("", ListUsers).
        Summary("List Users").
        Description("Returns a paginated list of users").
        Tags("Users").
        Register(usersGroup)

    framework.GET("/{id}", GetUser).
        Summary("Get User").
        Description("Retrieves a specific user by ID").
        Tags("Users").
        Register(usersGroup)

    // Register OpenAPI documentation
    openapi := openapi.NewOpenApi(app)
    openapi.RegisterOpenAPIDocs(
        "User API",
        "A type-safe REST API with validation",
        "1.0.0",
        "/openapi.json",
        "/docs",
    )

    fmt.Println("Server starting on :8080")
    fmt.Println("API Documentation: http://localhost:8080/docs")
    log.Fatal(http.ListenAndServe(":8080", app))
}
```

## Advanced Features Example

Here's an example combining query arrays and file uploads:

```go
package main

import (
    "context"
    "fmt"
    "io"
    "net/http"

    "github.com/example/api-framework/framework"
)

// Request with both query arrays and file upload
type ProcessImagesRequest struct {
    APIKey     string               `header:"X-API-Key" validate:"required"`
    Tags       []string             `query:"tags" doc:"Image tags for categorization"`
    Categories []string             `query:"category" doc:"Image categories"`
    Image      framework.FileField `form:"image" validate:"required" doc:"Image file to process"`
}

type ProcessImagesResponse struct {
    Success    bool     `json:"success"`
    ImageID    string   `json:"image_id"`
    Tags       []string `json:"tags"`
    Categories []string `json:"categories"`
    Size       int64    `json:"size"`
}

func ProcessImages(ctx context.Context, req ProcessImagesRequest) (ProcessImagesResponse, error) {
    defer req.Image.Content.Close()

    // Read the image
    imageData, err := io.ReadAll(req.Image.Content)
    if err != nil {
        return ProcessImagesResponse{}, fmt.Errorf("failed to read image: %w", err)
    }

    // Validate content type
    contentType := http.DetectContentType(imageData)
    if contentType != "image/jpeg" && contentType != "image/png" {
        return ProcessImagesResponse{}, fmt.Errorf("only JPEG and PNG images are supported")
    }

    // Process image with tags and categories
    imageID := generateImageID()

    // Save with metadata
    // saveImageWithMetadata(imageID, imageData, req.Tags, req.Categories)

    return ProcessImagesResponse{
        Success:    true,
        ImageID:    imageID,
        Tags:       req.Tags,
        Categories: req.Categories,
        Size:       req.Image.Size,
    }, nil
}

func main() {
    app := framework.New()

    framework.POST("/images/process", ProcessImages).
        Summary("Process Image").
        Description("Upload and process an image with tags and categories").
        Tags("Images").
        Register(app)

    http.ListenAndServe(":8080", app)
}
```

**Usage:**
```bash
curl -X POST "http://localhost:8080/images/process?tags=nature&tags=landscape&category=outdoor&category=scenic" \
  -H "X-API-Key: secret-key" \
  -F "image=@photo.jpg"
```

**Response:**
```json
{
  "success": true,
  "image_id": "img_abc123",
  "tags": ["nature", "landscape"],
  "categories": ["outdoor", "scenic"],
  "size": 2458624
}
```

## Testing Example

Testing your handlers is simple since they're just functions:

```go
func TestCreateUser(t *testing.T) {
    req := CreateUserRequest{}
    req.APIKey = "secret-key"
    req.Body.Name = "John Doe"
    req.Body.Email = "john@example.com"
    req.Body.Age = 30

    ctx := context.Background()
    resp, err := CreateUser(ctx, req)

    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }

    if resp.User.Name != "John Doe" {
        t.Errorf("Expected name 'John Doe', got: %s", resp.User.Name)
    }
}
```

## API Reference

### Core Types

```go
// Handler is a type-safe handler function that receives request context
type Handler[Req any, Resp any] func(ctx context.Context, req Req) (Resp, error)

// Middleware is a standard HTTP middleware
type Middleware func(next http.Handler) http.Handler

// NoRequest is used for handlers with no request parameters
type NoRequest struct{}

// FileField represents an uploaded file
type FileField struct {
    Filename string                // Original filename from client
    Size     int64                 // File size in bytes
    Header   *multipart.FileHeader // Full multipart header
    Content  io.ReadCloser         // File content stream
}

// FileUpload is a marker interface for file upload types
// Custom file types can implement this interface to be recognized as file uploads
type FileUpload interface {
    isFileUpload()
}
```

### Request Tags

**Supported Tags:**
- `route:"name"` - Bind to route parameter (e.g., `/users/{id}`)
- `query:"name"` - Bind to query parameter (e.g., `?page=1`)
- `header:"Name"` - Bind to HTTP header
- `body:""` - Bind to JSON request body
- `form:"name"` - Bind to multipart form field (file uploads)
- `validate:"rules"` - Validation rules (go-playground/validator)
- `doc:"description"` - Documentation for OpenAPI generation
- `json:"name"` - JSON field name (used with `body` tag)

**Type Support:**
- Scalar types: `string`, `int`, `int32`, `int64`, `uint`, `uint32`, `uint64`, `bool`, `float32`, `float64`
- Array types: `[]string`, `[]int`, `[]int64`, etc. (for query arrays)
- File types: `framework.FileField` (for file uploads)
- Struct types: Custom structs (for request body)

### Framework Methods

```go
// Create a new framework instance
func New() *Framework

// Create a route group with prefix
func (f *Framework) Group(prefix string) *Group

// Implement http.Handler
func (f *Framework) ServeHTTP(w http.ResponseWriter, r *http.Request)

// Get registered endpoints (for OpenAPI generation)
func (f *Framework) GetEndpoints() []*EndpointSpec
```

### Group Methods

```go
// Add middleware to the group
func (g *Group) Use(middleware ...Middleware) *Group

// Create a sub-group
func (g *Group) Group(prefix string) *Group
```

### Endpoint Builder Methods

```go
// HTTP method functions (GET, POST, PUT, PATCH, DELETE)
func GET[Req, Resp any](path string, handler Handler[Req, Resp]) *EndpointBuilder[Req, Resp]

// Fluent API methods
func (b *EndpointBuilder) Summary(summary string) *EndpointBuilder
func (b *EndpointBuilder) Description(description string) *EndpointBuilder
func (b *EndpointBuilder) Tags(tags ...string) *EndpointBuilder
func (b *EndpointBuilder) Use(middleware ...Middleware) *EndpointBuilder
func (b *EndpointBuilder) Register(router Router) error
```

## Performance

The framework is designed for performance:

- **Pre-computed Parsers**: Request parsing logic is computed at registration time, not per request
- **Minimal Reflection**: Reflection only used during registration, not in hot path
- **Zero Allocations**: Optimized field setters avoid unnecessary allocations
- **Native ServeMux**: Uses Go's standard HTTP multiplexer (Go 1.22+ path patterns)

## Requirements

- Go 1.22 or higher (for enhanced ServeMux path patterns)
- github.com/go-playground/validator/v10

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
