package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/RottenNinja-Go/framework"
	"github.com/RottenNinja-Go/framework/handler"
	"github.com/RottenNinja-Go/framework/openapi"
)

// User represents a user entity
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

// CreateUserRequest represents the request for creating a user
type CreateUserRequest struct {
	Header struct {
		APIKey      string `json:"X-API-Key" validate:"required" doc:"API authentication key"`
		ContentType string `json:"Content-Type" validate:"required,eq=application/json" doc:"Must be application/json"`
	}

	Body struct {
		Name  string `json:"name" validate:"required,min=3,max=50" doc:"User's full name"`
		Email string `json:"email" validate:"required,email" doc:"User's email address"`
		Age   int    `json:"age" validate:"required,min=18,max=120" doc:"User's age (must be 18 or older)"`
	}
}

// CreateUserResponse represents the response for creating a user
type CreateUserResponse struct {
	User User `json:"user"`
}

// GetUserRequest represents the request for getting a user
type GetUserRequest struct {
	Route struct {
		UserID string `json:"id" validate:"required" doc:"User ID"`
	}
	Header struct {
		APIKey string `json:"X-API-Key" validate:"required" doc:"API authentication key"`
	}
	Query struct {
		IncludeDetails bool `json:"include_details" doc:"Include detailed user information"`
	}
}

// GetUserResponse represents the response for getting a user
type GetUserResponse struct {
	User     User                   `json:"user"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateUserRequest represents the request for updating a user
type UpdateUserRequest struct {
	Route struct {
		UserID string `json:"id" validate:"required" doc:"User ID"`
	}
	Header struct {
		APIKey string `json:"X-API-Key" validate:"required" doc:"API authentication key"`
	}

	Body struct {
		Name  string `json:"name,omitempty" validate:"omitempty,min=3,max=50" doc:"User's full name"`
		Email string `json:"email,omitempty" validate:"omitempty,email" doc:"User's email address"`
		Age   int    `json:"age,omitempty" validate:"omitempty,min=18,max=120" doc:"User's age"`
	}
}

// UpdateUserResponse represents the response for updating a user
type UpdateUserResponse struct {
	User User `json:"user"`
}

// DeleteUserRequest represents the request for deleting a user
type DeleteUserRequest struct {
	Route struct {
		UserID string `json:"id" validate:"required" doc:"User ID"`
	}
	Header struct {
		APIKey string `json:"X-API-Key" validate:"required" doc:"API authentication key"`
	}
	Query struct {
		Force bool `json:"force" doc:"Force delete even if user has dependencies"`
	}
}

// DeleteUserResponse represents an empty response (will return 204)
type DeleteUserResponse struct{}

// ListUsersRequest represents the request for listing users
type ListUsersRequest struct {
	Header struct {
		APIKey string `json:"X-API-Key" validate:"required" doc:"API authentication key"`
	}
	Query struct {
		Page     int      `json:"page" validate:"omitempty,min=1" doc:"Page number (default: 1)"`
		PageSize int      `json:"page_size" validate:"omitempty,min=1,max=100" doc:"Items per page (default: 10, max: 100)"`
		SortBy   string   `json:"sort_by" validate:"omitempty,oneof=name email age created_at" doc:"Field to sort by"`
		Order    string   `json:"order" validate:"omitempty,oneof=asc desc" doc:"Sort order (asc or desc)"`
		Tags     []string `json:"tags" doc:"Filter by tags (can specify multiple: ?tags=admin&tags=premium)"`
	}
}

// ListUsersResponse represents the response for listing users
type ListUsersResponse struct {
	Users      []User                 `json:"users"`
	Pagination map[string]interface{} `json:"pagination"`
}

// UploadAvatarRequest represents the request for uploading a user avatar
type UploadAvatarRequest struct {
	Route struct {
		UserID string `json:"id" validate:"required" doc:"User ID"`
	}
	Header struct {
		APIKey string `json:"X-API-Key" validate:"required" doc:"API authentication key"`
	}
	Form struct {
		Avatar framework.FileField `json:"avatar" validate:"required" doc:"Avatar image file"`
	}
}

// UploadAvatarResponse represents the response for uploading an avatar
type UploadAvatarResponse struct {
	Message  string `json:"message"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// In-memory user storage (for demonstration)
var users = make(map[string]User)
var userCounter = 0

// LoggingMiddleware logs each request
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%s] %s %s\n", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware simulates authentication checking
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

func main() {
	// Create framework instance
	app := framework.New()

	// Create a /api/v1 group with logging middleware
	api := app.Group("/api/v1").Use(LoggingMiddleware)

	// Create a /users group under /api/v1 with auth middleware
	// This will have both logging (from parent) and auth middleware
	users := api.Group("/users").Use(AuthMiddleware)

	// Register endpoints using the group - all routes will be prefixed with /api/v1/users
	// and will have both logging and auth middleware applied

	// POST /api/v1/users - Create a new user
	handler.POST(users, "", CreateUser, func(eo handler.EndpointOptions) {
		eo.SetSummary("Create User")
		eo.SetDescription("Creates a new user with the provided information. Requires valid API key and user data.")
		eo.SetTags("Users")
	})

	// GET /api/v1/users - List users
	handler.GET(users, "", ListUsers, func(eo handler.EndpointOptions) {
		eo.SetSummary("List Users")
		eo.SetDescription("Retrieves a paginated list of users. Supports sorting and filtering.")
		eo.SetTags("Users")
	})

	// GET /api/v1/users/{id} - Get a specific user
	handler.GET(users, "/{id}", GetUser, func(eo handler.EndpointOptions) {
		eo.SetSummary("Get User")
		eo.SetDescription("Retrieves a specific user by their ID. Optionally includes detailed information.")
		eo.SetTags("Users")
	})

	// PUT /api/v1/users/{id} - Update a user
	handler.PUT(users, "/{id}", UpdateUser, func(eo handler.EndpointOptions) {
		eo.SetSummary("Update User")
		eo.SetDescription("Updates an existing user's information. All fields are optional.")
		eo.SetTags("Users")
	})

	// DELETE /api/v1/users/{id} - Delete a user
	handler.DELETE(users, "/{id}", DeleteUser, func(eo handler.EndpointOptions) {
		eo.SetSummary("Delete User")
		eo.SetDescription("Deletes a user by their ID. Use force=true to override dependency checks.")
		eo.SetTags("Users")
	})

	// POST /api/v1/users/{id}/avatar - Upload user avatar
	handler.POST(users, "/{id}/avatar", UploadAvatar, func(eo handler.EndpointOptions) {
		eo.SetSummary("Upload User Avatar")
		eo.SetDescription("Uploads an avatar image for a user. Accepts multipart/form-data with 'avatar' field.")
		eo.SetTags("Users")
	})

	// Register OpenAPI documentation endpoints on the root (not in the group)
	openapi := openapi.NewOpenApi(app)
	openapi.RegisterOpenAPIDocs(
		"User API (Type-Safe)",
		"A fully type-safe API with compile-time guarantees using Go generics and route grouping",
		"1.0.0",
		"/openapi.json",
		"/docs",
	)

	// Start server
	fmt.Println("🚀 Type-Safe API Server starting on :8080")
	fmt.Println("📚 API Documentation: http://localhost:8080/docs")
	fmt.Println("📄 OpenAPI Spec: http://localhost:8080/openapi.json")
	fmt.Println("")
	fmt.Println("✨ This API uses Go generics for compile-time type safety!")
	fmt.Println("🔗 Route Groups: All user endpoints are under /api/v1/users")
	fmt.Println("🔒 Middleware: Logging on /api/v1/*, Auth on /api/v1/users/*")
	fmt.Println("📦 Query Arrays: ?tags=admin&tags=premium (multiple values)")
	fmt.Println("📁 File Uploads: POST /api/v1/users/{id}/avatar with multipart/form-data")
	log.Fatal(http.ListenAndServe(":8080", app))
}

// CreateUser handles user creation with full type safety
func CreateUser(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
	// Validate API key
	if req.Header.APIKey != "secret-key" {
		return CreateUserResponse{}, fmt.Errorf("invalid API key")
	}

	// Create user
	userCounter++
	user := User{
		ID:    fmt.Sprintf("user-%d", userCounter),
		Name:  req.Body.Name,
		Email: req.Body.Email,
		Age:   req.Body.Age,
	}

	users[user.ID] = user

	// Return type-safe response
	return CreateUserResponse{User: user}, nil
}

// GetUser handles getting a specific user with type safety
func GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	// Validate API key
	if req.Header.APIKey != "secret-key" {
		return GetUserResponse{}, fmt.Errorf("invalid API key")
	}

	user, exists := users[req.Route.UserID]
	if !exists {
		return GetUserResponse{}, fmt.Errorf("user not found")
	}

	response := GetUserResponse{
		User: user,
	}

	// If include_details is true, add extra information
	if req.Query.IncludeDetails {
		response.Metadata = map[string]interface{}{
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
		}
	}

	return response, nil
}

// UpdateUser handles updating a user with type safety
func UpdateUser(ctx context.Context, req UpdateUserRequest) (UpdateUserResponse, error) {
	// Validate API key
	if req.Header.APIKey != "secret-key" {
		return UpdateUserResponse{}, fmt.Errorf("invalid API key")
	}

	user, exists := users[req.Route.UserID]
	if !exists {
		return UpdateUserResponse{}, fmt.Errorf("user not found")
	}

	// Update fields if provided
	if req.Body.Name != "" {
		user.Name = req.Body.Name
	}
	if req.Body.Email != "" {
		user.Email = req.Body.Email
	}
	if req.Body.Age != 0 {
		user.Age = req.Body.Age
	}

	users[req.Route.UserID] = user

	return UpdateUserResponse{User: user}, nil
}

// DeleteUser handles deleting a user with type safety
func DeleteUser(ctx context.Context, req DeleteUserRequest) (DeleteUserResponse, error) {
	// Validate API key
	if req.Header.APIKey != "secret-key" {
		return DeleteUserResponse{}, fmt.Errorf("invalid API key")
	}

	_, exists := users[req.Route.UserID]
	if !exists {
		return DeleteUserResponse{}, fmt.Errorf("user not found")
	}

	delete(users, req.Route.UserID)

	// Return empty response (will result in 204 No Content)
	return DeleteUserResponse{}, nil
}

// ListUsers handles listing users with pagination and type safety
func ListUsers(ctx context.Context, req ListUsersRequest) (ListUsersResponse, error) {
	// Validate API key
	if req.Header.APIKey != "secret-key" {
		return ListUsersResponse{}, fmt.Errorf("invalid API key")
	}

	// Set defaults
	page := req.Query.Page
	if page == 0 {
		page = 1
	}

	pageSize := req.Query.PageSize
	if pageSize == 0 {
		pageSize = 10
	}

	// Convert map to slice
	userList := make([]User, 0, len(users))
	for _, user := range users {
		userList = append(userList, user)
	}

	// Demonstrate query array usage - log the tags if provided
	if len(req.Query.Tags) > 0 {
		fmt.Printf("Filtering users by tags: %v\n", req.Query.Tags)
	}

	return ListUsersResponse{
		Users: userList,
		Pagination: map[string]interface{}{
			"page":        page,
			"page_size":   pageSize,
			"total":       len(userList),
			"total_pages": (len(userList) + pageSize - 1) / pageSize,
			"tags":        req.Query.Tags,
		},
	}, nil
}

// UploadAvatar handles uploading a user avatar with file upload support
func UploadAvatar(ctx context.Context, req UploadAvatarRequest) (UploadAvatarResponse, error) {
	// Validate API key
	if req.Header.APIKey != "secret-key" {
		return UploadAvatarResponse{}, fmt.Errorf("invalid API key")
	}

	// Check if user exists
	_, exists := users[req.Route.UserID]
	if !exists {
		return UploadAvatarResponse{}, fmt.Errorf("user not found")
	}

	// Close the file when done reading (important!)
	defer req.Form.Avatar.Content.Close()

	// In a real application, you would:
	// 1. Validate file type (image/jpeg, image/png, etc.)
	// 2. Validate file size
	// 3. Save the file to storage (S3, local disk, etc.)
	// 4. Update the user record with the avatar URL

	fmt.Printf("Received avatar upload: %s (%d bytes)\n", req.Form.Avatar.Filename, req.Form.Avatar.Size)

	return UploadAvatarResponse{
		Message:  "Avatar uploaded successfully",
		Filename: req.Form.Avatar.Filename,
		Size:     req.Form.Avatar.Size,
	}, nil
}
