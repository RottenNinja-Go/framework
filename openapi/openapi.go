package openapi

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/RottenNinja-Go/framework"
)

// OpenAPISpec represents the OpenAPI 3.0 specification
type OpenAPISpec struct {
	OpenAPI    string              `json:"openapi"`
	Info       OpenAPIInfo         `json:"info"`
	Paths      map[string]PathItem `json:"paths"`
	Components *Components         `json:"components,omitempty"`
}

// OpenAPIInfo represents API information
type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// PathItem represents operations available on a single path
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

// Operation describes a single API operation
type Operation struct {
	Summary     string                     `json:"summary,omitempty"`
	Description string                     `json:"description,omitempty"`
	Tags        []string                   `json:"tags,omitempty"`
	Parameters  []Parameter                `json:"parameters,omitempty"`
	RequestBody *RequestBody               `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses"`
}

// Parameter describes a single operation parameter
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // query, header, path
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required"`
	Schema      *Schema `json:"schema"`
}

// RequestBody describes a single request body
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required"`
	Content     map[string]MediaType `json:"content"`
}

// MediaType provides schema and examples for the media type
type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

// OpenAPIResponse describes a single response in OpenAPI spec
type OpenAPIResponse struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// Schema represents a data type
type Schema struct {
	Type       string             `json:"type,omitempty"`
	Format     string             `json:"format,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	Required   []string           `json:"required,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
	Ref        string             `json:"$ref,omitempty"`
	Enum       []interface{}      `json:"enum,omitempty"`
	Minimum    *float64           `json:"minimum,omitempty"`
	Maximum    *float64           `json:"maximum,omitempty"`
	MinLength  *int               `json:"minLength,omitempty"`
	MaxLength  *int               `json:"maxLength,omitempty"`
	Pattern    string             `json:"pattern,omitempty"`
}

// Components holds reusable objects
type Components struct {
	Schemas map[string]*Schema `json:"schemas,omitempty"`
}

func NewOpenApi(f *framework.Framework) *OpenApi {
	return &OpenApi{f: f}
}

type OpenApi struct {
	f *framework.Framework
}

// GenerateOpenAPI generates OpenAPI specification
func (f *OpenApi) GenerateOpenAPI(title, description, version string) *OpenAPISpec {
	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:       title,
			Description: description,
			Version:     version,
		},
		Paths: make(map[string]PathItem),
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}

	// Generate paths from endpoints
	for _, endpoint := range f.f.GetEndpoints() {
		pathItem, ok := spec.Paths[endpoint.Path]
		if !ok {
			pathItem = PathItem{}
		}

		operation := f.generateOperation(endpoint, spec.Components.Schemas)

		switch endpoint.Method {
		case "GET":
			pathItem.Get = operation
		case "POST":
			pathItem.Post = operation
		case "PUT":
			pathItem.Put = operation
		case "PATCH":
			pathItem.Patch = operation
		case "DELETE":
			pathItem.Delete = operation
		}

		spec.Paths[endpoint.Path] = pathItem
	}

	return spec
}

// generateOperation generates an Operation from an EndpointSpec
func (f *OpenApi) generateOperation(endpoint *framework.EndpointSpec, schemas map[string]*Schema) *Operation {
	// Generate response schema from the actual response type
	responseSchema := f.reflectTypeToSchemaExpanded(endpoint.ResponseType)

	operation := &Operation{
		Summary:     endpoint.Summary,
		Description: endpoint.Description,
		Tags:        endpoint.Tags,
		Parameters:  make([]Parameter, 0),
		Responses: map[string]OpenAPIResponse{
			"200": {
				Description: "Successful response",
				Content: map[string]MediaType{
					"application/json": {
						Schema: responseSchema,
					},
				},
			},
			"400": {
				Description: "Bad request - validation error",
				Content: map[string]MediaType{
					"application/json": {
						Schema: f.getErrorSchema(),
					},
				},
			},
			"500": {
				Description: "Internal server error",
				Content: map[string]MediaType{
					"application/json": {
						Schema: f.getErrorSchema(),
					},
				},
			},
		},
	}

	// Parse request type
	reqType := endpoint.RequestType

	// Track form fields for multipart/form-data
	formFields := make(map[string]*Schema)
	formFieldsRequired := make([]string, 0)
	hasFormFields := false

	for i := 0; i < reqType.NumField(); i++ {
		field := reqType.Field(i)

		// Parse headers
		if headerTag := field.Tag.Get("header"); headerTag != "" {
			param := Parameter{
				Name:        headerTag,
				In:          "header",
				Description: field.Tag.Get("doc"),
				Required:    strings.Contains(field.Tag.Get("validate"), "required"),
				Schema:      f.reflectTypeToSchema(field.Type),
			}
			operation.Parameters = append(operation.Parameters, param)
		}

		// Parse route params
		if routeTag := field.Tag.Get("route"); routeTag != "" {
			param := Parameter{
				Name:        routeTag,
				In:          "path",
				Description: field.Tag.Get("doc"),
				Required:    true, // Path params are always required
				Schema:      f.reflectTypeToSchema(field.Type),
			}
			operation.Parameters = append(operation.Parameters, param)
		}

		// Parse query params
		if queryTag := field.Tag.Get("query"); queryTag != "" {
			param := Parameter{
				Name:        queryTag,
				In:          "query",
				Description: field.Tag.Get("doc"),
				Required:    strings.Contains(field.Tag.Get("validate"), "required"),
				Schema:      f.reflectTypeToSchema(field.Type),
			}
			operation.Parameters = append(operation.Parameters, param)
		}

		// Parse form fields (for file uploads and multipart data)
		if formTag := field.Tag.Get("form"); formTag != "" {
			hasFormFields = true

			// Check if this field implements the FileUpload interface
			fileUploadInterface := reflect.TypeOf((*framework.FileUpload)(nil)).Elem()
			isFileField := field.Type.Implements(fileUploadInterface)

			var fieldSchema *Schema
			if isFileField {
				// File upload field
				fieldSchema = &Schema{
					Type:   "string",
					Format: "binary",
				}
			} else {
				// Regular form field
				fieldSchema = f.reflectTypeToSchema(field.Type)
			}

			formFields[formTag] = fieldSchema

			if strings.Contains(field.Tag.Get("validate"), "required") {
				formFieldsRequired = append(formFieldsRequired, formTag)
			}
		}

		// Parse body
		if _, hasBodyTag := field.Tag.Lookup("body"); hasBodyTag {
			bodySchema := f.structToSchema(field.Type, schemas)
			operation.RequestBody = &RequestBody{
				Description: field.Tag.Get("doc"),
				Required:    strings.Contains(field.Tag.Get("validate"), "required"),
				Content: map[string]MediaType{
					"application/json": {
						Schema: bodySchema,
					},
				},
			}
		}
	}

	// If form fields were found, create multipart/form-data request body
	if hasFormFields {
		operation.RequestBody = &RequestBody{
			Description: "Multipart form data",
			Required:    len(formFieldsRequired) > 0,
			Content: map[string]MediaType{
				"multipart/form-data": {
					Schema: &Schema{
						Type:       "object",
						Properties: formFields,
						Required:   formFieldsRequired,
					},
				},
			},
		}
	}

	return operation
}

// reflectTypeToSchema converts a reflect.Type to a Schema
// This function does NOT expand struct properties - use structToSchema for that
func (f *OpenApi) reflectTypeToSchema(t reflect.Type) *Schema {
	schema := &Schema{}

	switch t.Kind() {
	case reflect.String:
		schema.Type = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = "integer"
		if t.Kind() == reflect.Int64 {
			schema.Format = "int64"
		} else {
			schema.Format = "int32"
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = "integer"
		schema.Minimum = new(float64)
		*schema.Minimum = 0
	case reflect.Bool:
		schema.Type = "boolean"
	case reflect.Float32:
		schema.Type = "number"
		schema.Format = "float"
	case reflect.Float64:
		schema.Type = "number"
		schema.Format = "double"
	case reflect.Slice, reflect.Array:
		schema.Type = "array"
		schema.Items = f.reflectTypeToSchemaExpanded(t.Elem())
	case reflect.Struct:
		schema.Type = "object"
	default:
		schema.Type = "string"
	}

	return schema
}

// reflectTypeToSchemaExpanded converts a reflect.Type to a Schema, expanding struct properties
func (f *OpenApi) reflectTypeToSchemaExpanded(t reflect.Type) *Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// For structs, expand the properties
	if t.Kind() == reflect.Struct {
		return f.structToSchemaInternal(t)
	}

	// For other types, use the simple conversion
	return f.reflectTypeToSchema(t)
}

// structToSchemaInternal is an internal version that doesn't take the schemas map
// This is used for nested structs where we don't need schema references
func (f *OpenApi) structToSchemaInternal(t reflect.Type) *Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return f.reflectTypeToSchema(t)
	}

	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		Required:   make([]string, 0),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON tag name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
		}

		// Create field schema - use expanded version for nested structs
		var fieldSchema *Schema
		if field.Type.Kind() == reflect.Struct || (field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct) {
			fieldSchema = f.reflectTypeToSchemaExpanded(field.Type)
		} else if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Struct {
			fieldSchema = f.reflectTypeToSchema(field.Type) // This will handle the array with expanded items
		} else if field.Type.Kind() == reflect.Map {
			// Handle map[string]interface{} and similar types
			fieldSchema = &Schema{
				Type: "object",
			}
			// If we want to be more specific about the value type, we could inspect field.Type.Elem()
		} else {
			fieldSchema = f.reflectTypeToSchema(field.Type)
		}

		// Add validation info from tags
		if validateTag := field.Tag.Get("validate"); validateTag != "" {
			f.applyValidationToSchema(fieldSchema, validateTag)

			if strings.Contains(validateTag, "required") {
				schema.Required = append(schema.Required, fieldName)
			}
		}

		schema.Properties[fieldName] = fieldSchema
	}

	return schema
}

// structToSchema converts a struct type to a Schema
func (f *OpenApi) structToSchema(t reflect.Type, schemas map[string]*Schema) *Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return f.reflectTypeToSchema(t)
	}

	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		Required:   make([]string, 0),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON tag name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
		}

		// Create field schema
		fieldSchema := f.reflectTypeToSchema(field.Type)

		// Add validation info from tags
		if validateTag := field.Tag.Get("validate"); validateTag != "" {
			f.applyValidationToSchema(fieldSchema, validateTag)

			if strings.Contains(validateTag, "required") {
				schema.Required = append(schema.Required, fieldName)
			}
		}

		// Add documentation
		if doc := field.Tag.Get("doc"); doc != "" {
			// OpenAPI doesn't support description in schema properties directly,
			// but we can add it as a custom field if needed
		}

		schema.Properties[fieldName] = fieldSchema
	}

	return schema
}

// applyValidationToSchema applies validation rules to schema
func (f *OpenApi) applyValidationToSchema(schema *Schema, validateTag string) {
	rules := strings.Split(validateTag, ",")

	for _, rule := range rules {
		parts := strings.Split(rule, "=")
		ruleName := strings.TrimSpace(parts[0])

		switch ruleName {
		case "min":
			if len(parts) > 1 {
				if schema.Type == "string" {
					var minLen int
					fmt.Sscanf(parts[1], "%d", &minLen)
					schema.MinLength = &minLen
				} else if schema.Type == "number" || schema.Type == "integer" {
					var min float64
					fmt.Sscanf(parts[1], "%f", &min)
					schema.Minimum = &min
				}
			}
		case "max":
			if len(parts) > 1 {
				if schema.Type == "string" {
					var maxLen int
					fmt.Sscanf(parts[1], "%d", &maxLen)
					schema.MaxLength = &maxLen
				} else if schema.Type == "number" || schema.Type == "integer" {
					var max float64
					fmt.Sscanf(parts[1], "%f", &max)
					schema.Maximum = &max
				}
			}
		case "email":
			schema.Format = "email"
		case "url":
			schema.Format = "uri"
		}
	}
}

// getErrorSchema returns the schema for error responses
func (f *OpenApi) getErrorSchema() *Schema {
	return &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"error": {
				Type: "string",
			},
			"details": {
				Type: "object",
				Properties: map[string]*Schema{
					"*": {Type: "string"},
				},
			},
		},
		Required: []string{"error"},
	}
}

// SwaggerUIResponse is a custom response that returns HTML
type SwaggerUIResponse struct {
	html string
}

// WriteResponse implements the Responder interface for Swagger UI
func (r SwaggerUIResponse) WriteResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(r.html))
}

// RegisterOpenAPIDocs registers the OpenAPI spec and Swagger UI endpoints
// This should be called after all other endpoints are registered
func (f *OpenApi) RegisterOpenAPIDocs(title, description, version, specPath, docsPath string) error {
	// Register OpenAPI spec endpoint
	specHandler := func(ctx context.Context, _ framework.NoRequest) (*OpenAPISpec, error) {
		spec := f.GenerateOpenAPI(title, description, version)
		return spec, nil
	}

	if err := framework.GET(specPath, specHandler).
		Summary("OpenAPI Specification").
		Description("Returns the OpenAPI 3.0 specification for this API").
		Tags("Documentation").
		Register(f.f); err != nil {
		return fmt.Errorf("failed to register OpenAPI spec endpoint: %w", err)
	}

	// Register Swagger UI endpoint
	uiHandler := func(ctx context.Context, _ framework.NoRequest) (SwaggerUIResponse, error) {
		html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.10.0/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.10.0/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.10.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: "%s",
                dom_id: '#swagger-ui',
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                layout: "BaseLayout"
            });
        };
    </script>
</body>
</html>`, specPath)

		return SwaggerUIResponse{html: html}, nil
	}

	if err := framework.GET(docsPath, uiHandler).
		Summary("API Documentation").
		Description("Interactive API documentation using Swagger UI").
		Tags("Documentation").
		Register(f.f); err != nil {
		return fmt.Errorf("failed to register Swagger UI endpoint: %w", err)
	}

	return nil
}
