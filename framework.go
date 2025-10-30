package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// Framework is the main API framework
type Framework struct {
	mux       *http.ServeMux
	validator *validator.Validate
	endpoints []*EndpointSpec
}

// Group represents a group of routes with a common path prefix and middleware
type Group struct {
	framework   *Framework
	prefix      string
	middlewares []Middleware
}

type StatusResponse[Resp any] struct {
	Code int
	Body Resp
}

// Router is an interface that both Framework and Group implement
// This allows the same GET, POST, PUT, PATCH, DELETE functions to work with both
type Router interface {
	getFramework() *Framework
	getPrefix() string
	getMiddlewares() []Middleware
}

type NoRequest struct{}

// FileUpload is a marker interface for types that represent uploaded files
// Any type implementing this interface will be treated as a file upload field
type FileUpload interface {
	isFileUpload()
}

// FileField represents an uploaded file
type FileField struct {
	Filename string
	Size     int64
	Header   *multipart.FileHeader
	Content  io.ReadCloser
}

// isFileUpload implements the FileUpload interface
func (FileField) isFileUpload() {}

// Handler is a type-safe handler function that takes a request and returns a response
type Handler[Req any, Resp any] func(ctx context.Context, req Req) (Resp, error)

// Middleware is a standard HTTP middleware function
// It receives the next handler and returns a new handler that can wrap it
type Middleware func(next http.Handler) http.Handler

//	type HandlerRoute[TReq any, TResp any] struct {
//		*EndpointSpec
//		Handler func(ctx context.Context, req TReq) (TResp, error)
//	}
type Endpoint interface {
	SetSummary(summary string)
	SetDescription(description string)
	Use(middleware ...Middleware)
	SetTags(tags ...string)
	getSpec() *EndpointSpec
}

// EndpointSpec defines the specification for an endpoint
type EndpointSpec struct {
	Method       string
	RelativePath string
	FullPath     string
	Summary      string
	Description  string
	Tags         []string
	RequestType  reflect.Type
	ResponseType reflect.Type
	Middlewares  []Middleware

	AllMiddlewares []Middleware
	handlerPrepFn  func(*Framework) http.HandlerFunc
	// handlerFunc  http.HandlerFunc
}

// Summary sets the endpoint summary
func (b *EndpointSpec) SetSummary(summary string) {
	b.Summary = summary
}

// Description sets the endpoint description
func (b *EndpointSpec) SetDescription(description string) {
	b.Description = description
}

// Tags sets the endpoint tags
func (b *EndpointSpec) SetTags(tags ...string) {
	b.Tags = tags
}

// Use adds one or more middleware functions to the endpoint
// Middleware is applied in the order it's added (first added = outermost wrapper)
// Example: Use(logging, auth, ratelimit) wraps as logging(auth(ratelimit(handler)))
func (b *EndpointSpec) Use(middleware ...Middleware) {
	b.Middlewares = append(b.Middlewares, middleware...)
}
func (b *EndpointSpec) getSpec() *EndpointSpec {
	return b
}

// fieldParser holds pre-computed parsing logic for a field
type fieldParser struct {
	fieldIndex       int
	nestedFieldIndex int  // Index of the field within the nested struct
	isNested         bool // True if this field is nested within Route/Header/Query/Form/Body
	fieldType        reflect.Type
	fieldKind        reflect.Kind

	// Parsing configuration
	sourceType string // "header", "route", "query", "body", "form"
	sourceName string // The name of the header/route/query/form parameter

	// Pre-computed setter function (avoids reflection on hot path)
	setter      func(fieldValue reflect.Value, strValue string) error
	isSlice     bool // True if this field is a slice (for query arrays)
	isFileField bool // True if this field is a FileField (for file uploads)
}

// requestParser holds all pre-computed parsing logic for a request type
type requestParser struct {
	requestType  reflect.Type
	fieldParsers []fieldParser
	hasBodyField bool
	bodyFieldIdx int
}

// Responder is an interface for custom responses that need control over status codes and headers
type Responder interface {
	WriteResponse(w http.ResponseWriter)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// ValidationError represents a validation error for a specific field
type ValidationError struct {
	Field      string   `json:"field"`
	SourceType string   `json:"source_type,omitempty"` // "header", "query", "route", "body"
	Errors     []string `json:"errors"`
}

// ValidationErrorResponse represents a validation error response
type ValidationErrorResponse struct {
	Error  string            `json:"error"`
	Fields []ValidationError `json:"fields,omitempty"`
}

// validationErrorWrapper wraps validation errors to implement the error interface
type validationErrorWrapper struct {
	validationErrors []ValidationError
}

func (v *validationErrorWrapper) Error() string {
	return "validation failed"
}

func (v *validationErrorWrapper) ValidationErrors() []ValidationError {
	return v.validationErrors
}

// New creates a new Framework instance
func New() *Framework {
	return &Framework{
		mux:       http.NewServeMux(),
		validator: validator.New(),
		endpoints: make([]*EndpointSpec, 0),
	}
}

// getFramework implements Router interface for Framework
func (f *Framework) getFramework() *Framework {
	return f
}

// getPrefix implements Router interface for Framework
func (f *Framework) getPrefix() string {
	return ""
}

// getMiddlewares implements Router interface for Framework
func (f *Framework) getMiddlewares() []Middleware {
	return nil
}

// Group creates a new route group with the given path prefix
// Example: api := app.Group("/api/v1")
func (f *Framework) Group(prefix string) *Group {
	return &Group{
		framework:   f,
		prefix:      prefix,
		middlewares: make([]Middleware, 0),
	}
}

// getFramework implements Router interface for Group
func (g *Group) getFramework() *Framework {
	return g.framework
}

// getPrefix implements Router interface for Group
func (g *Group) getPrefix() string {
	return g.prefix
}

// getMiddlewares implements Router interface for Group
func (g *Group) getMiddlewares() []Middleware {
	return g.middlewares
}

// Use adds middleware to the group
// All routes registered on this group will have this middleware applied
func (g *Group) Use(middleware ...Middleware) *Group {
	g.middlewares = append(g.middlewares, middleware...)
	return g
}

// Group creates a sub-group with an additional path prefix
// The new group inherits all middleware from the parent group
// Example: api := app.Group("/api"); v1 := api.Group("/v1")
func (g *Group) Group(prefix string) *Group {
	return &Group{
		framework:   g.framework,
		prefix:      g.prefix + prefix,
		middlewares: append([]Middleware{}, g.middlewares...), // Copy parent middlewares
	}
}

// registerWithMiddleware registers a new endpoint with type-safe handler and middleware
func CreateEndpoint[TReq any, TResp any](method, path string, handler func(ctx context.Context, req TReq) (TResp, error)) Endpoint {
	route := &EndpointSpec{
		Method:       method,
		RelativePath: path,
	}

	// Get request and response types for OpenAPI generation
	var reqExample TReq
	route.RequestType = reflect.TypeOf(reqExample)

	var respExample TResp
	route.ResponseType = reflect.TypeOf(respExample)

	// Build request parser plan at registration time (expensive reflection here)
	parser := buildRequestParser(route.RequestType)

	// Create HTTP handler function with pre-computed parser
	route.handlerPrepFn = func(f *Framework) http.HandlerFunc { return createTypeSafeHandler(f, handler, parser) }
	return route
}

func RegisterEndpoint(router Router, e Endpoint) {
	route := e.getSpec()

	f := router.getFramework()

	// Combine group middlewares with endpoint-specific middlewares
	// Group middlewares should be applied first (outermost)
	route.AllMiddlewares = append(router.getMiddlewares(), route.Middlewares...)

	// Combine group prefix with endpoint path
	route.FullPath = router.getPrefix() + route.RelativePath

	// Apply middleware in reverse order (so first middleware added is outermost)
	var finalHandler http.Handler = route.handlerPrepFn(f)
	for i := len(route.AllMiddlewares) - 1; i >= 0; i-- {
		finalHandler = route.AllMiddlewares[i](finalHandler)
	}
	// route.handlerFunc = finalHandler.ServeHTTP

	f.endpoints = append(f.endpoints, route)

	// Register with ServeMux using method and path pattern
	// Go 1.22+ supports patterns like "GET /users/{id}"
	pattern := route.Method + " " + route.FullPath
	f.mux.Handle(pattern, finalHandler)
}

// RegisterHandlerRoute registers a new endpoint with type-safe handler and middleware
func RegisterHandlerRoute[TReq any, TResp any](router Router, method, path string, handler func(ctx context.Context, req TReq) (TResp, error), callBackFn func(Endpoint)) {
	ep := CreateEndpoint(method, path, handler)
	callBackFn(ep)
	RegisterEndpoint(router, ep)
}

// buildRequestParser builds a pre-computed parser plan for a request type
// This function does all the expensive reflection work at registration time
func buildRequestParser(reqType reflect.Type) *requestParser {
	parser := &requestParser{
		requestType:  reqType,
		fieldParsers: make([]fieldParser, 0),
	}

	// Analyze each field in the request struct
	for i := 0; i < reqType.NumField(); i++ {
		field := reqType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fieldName := field.Name
		fieldKind := field.Type.Kind()

		// Check if this is a nested struct for Route, Header, Query, Form, or Body
		if fieldKind == reflect.Struct {
			switch fieldName {
			case "Route":
				parseNestedStruct(parser, field.Type, i, "route")
			case "Header":
				parseNestedStruct(parser, field.Type, i, "header")
			case "Query":
				parseNestedStruct(parser, field.Type, i, "query")
			case "Form":
				parseNestedStructForForm(parser, field.Type, i)
			case "Body":
				parser.hasBodyField = true
				parser.bodyFieldIdx = i
			}
		}
	}

	return parser
}

// parseNestedStruct parses a nested struct (Route, Header, Query) and extracts fields using json tags
func parseNestedStruct(parser *requestParser, structType reflect.Type, parentIndex int, sourceType string) {
	for j := 0; j < structType.NumField(); j++ {
		nestedField := structType.Field(j)

		// Skip unexported fields
		if !nestedField.IsExported() {
			continue
		}

		// Get the json tag for the field name
		jsonTag := nestedField.Tag.Get("json")
		if jsonTag == "" {
			// If no json tag, use the field name
			jsonTag = nestedField.Name
		}

		fieldKind := nestedField.Type.Kind()
		fieldType := nestedField.Type

		// Check if this is a slice
		isSlice := fieldKind == reflect.Slice

		// For slices, get the element type for the setter
		if isSlice {
			fieldKind = fieldType.Elem().Kind()
		}

		// Create pre-computed setter for this field type
		setter := createFieldSetter(fieldKind)

		parser.fieldParsers = append(parser.fieldParsers, fieldParser{
			fieldIndex:       parentIndex,
			nestedFieldIndex: j,
			fieldType:        nestedField.Type,
			fieldKind:        fieldKind,
			sourceType:       sourceType,
			sourceName:       jsonTag,
			setter:           setter,
			isSlice:          isSlice,
			isNested:         true,
		})
	}
}

// parseNestedStructForForm parses a nested Form struct for file uploads
func parseNestedStructForForm(parser *requestParser, structType reflect.Type, parentIndex int) {
	fileUploadInterface := reflect.TypeOf((*FileUpload)(nil)).Elem()

	for j := 0; j < structType.NumField(); j++ {
		nestedField := structType.Field(j)

		// Skip unexported fields
		if !nestedField.IsExported() {
			continue
		}

		// Get the json tag for the field name
		jsonTag := nestedField.Tag.Get("json")
		if jsonTag == "" {
			// If no json tag, use the field name
			jsonTag = nestedField.Name
		}

		// Check if this field implements the FileUpload interface
		isFileField := nestedField.Type.Implements(fileUploadInterface)

		parser.fieldParsers = append(parser.fieldParsers, fieldParser{
			fieldIndex:       parentIndex,
			nestedFieldIndex: j,
			fieldType:        nestedField.Type,
			fieldKind:        nestedField.Type.Kind(),
			sourceType:       "form",
			sourceName:       jsonTag,
			setter:           nil, // File fields don't use the setter
			isFileField:      isFileField,
			isNested:         true,
		})
	}
}

// createFieldSetter creates a type-specific setter function at registration time
// This avoids the switch statement in the hot path
func createFieldSetter(kind reflect.Kind) func(reflect.Value, string) error {
	switch kind {
	case reflect.String:
		return func(field reflect.Value, value string) error {
			field.SetString(value)
			return nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(field reflect.Value, value string) error {
			var intVal int64
			if _, err := fmt.Sscanf(value, "%d", &intVal); err != nil {
				return fmt.Errorf("invalid integer value")
			}
			field.SetInt(intVal)
			return nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return func(field reflect.Value, value string) error {
			var uintVal uint64
			if _, err := fmt.Sscanf(value, "%d", &uintVal); err != nil {
				return fmt.Errorf("invalid unsigned integer value")
			}
			field.SetUint(uintVal)
			return nil
		}
	case reflect.Bool:
		return func(field reflect.Value, value string) error {
			field.SetBool(value == "true" || value == "1")
			return nil
		}
	case reflect.Float32, reflect.Float64:
		return func(field reflect.Value, value string) error {
			var floatVal float64
			if _, err := fmt.Sscanf(value, "%f", &floatVal); err != nil {
				return fmt.Errorf("invalid float value")
			}
			field.SetFloat(floatVal)
			return nil
		}
	default:
		return func(field reflect.Value, value string) error {
			return fmt.Errorf("unsupported field type: %s", kind)
		}
	}
}

// createTypeSafeHandler creates an HTTP handler that parses and validates the request
// This is a top-level function because Go doesn't support generic methods
// The parser parameter contains pre-computed parsing logic, avoiding reflection on hot path
func createTypeSafeHandler[Req any, Resp any](f *Framework, handler Handler[Req, Resp], parser *requestParser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create new instance of request struct
		var req Req
		reqValue := reflect.ValueOf(&req).Elem()

		// Parse using pre-computed parser (fast path - minimal reflection)
		if err := f.parseWithPlan(r, reqValue, parser); err != nil {
			// Check if it's a validation error
			if validationErr, ok := err.(*validationErrorWrapper); ok {
				f.writeValidationError(w, http.StatusBadRequest, validationErr.ValidationErrors())
			} else {
				f.writeError(w, http.StatusBadRequest, err.Error(), nil)
			}
			return
		}

		// Call the type-safe handler
		response, err := handler(r.Context(), req)

		// Handle errors
		if err != nil {
			f.writeError(w, http.StatusInternalServerError, err.Error(), nil)
			return
		}

		// Write response
		writeResponse(w, response)
	}
}

// writeResponse writes the response to the HTTP response writer
func writeResponse[Resp any](w http.ResponseWriter, response Resp) {
	// Check if response implements Responder interface
	if responder, ok := any(response).(Responder); ok {
		responder.WriteResponse(w)
		return
	}

	if statusResponder, ok := any(response).(StatusResponse[Resp]); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusResponder.Code)
		json.NewEncoder(w).Encode(statusResponder.Body)
		return
	}

	// Default 200 OK response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// parseWithPlan parses the request using a pre-computed parser plan
// This is the OPTIMIZED hot path - uses pre-computed field parsers instead of reflection
func (f *Framework) parseWithPlan(r *http.Request, reqValue reflect.Value, parser *requestParser) error {
	// Iterate through pre-computed field parsers (no reflection needed for tag lookup!)
	for _, fp := range parser.fieldParsers {
		// Get the actual field value (either top-level or nested)
		var fieldValue reflect.Value
		if fp.isNested {
			// Get the parent struct (Route, Header, Query, or Form)
			parentField := reqValue.Field(fp.fieldIndex)
			// Get the nested field within the parent struct
			fieldValue = parentField.Field(fp.nestedFieldIndex)
		} else {
			// Old behavior for backward compatibility (should not be reached with new system)
			fieldValue = reqValue.Field(fp.fieldIndex)
		}

		// Handle file uploads
		if fp.sourceType == "form" && fp.isFileField {
			if err := f.parseFileField(r, fieldValue, fp.sourceName); err != nil {
				return fmt.Errorf("form '%s': %w", fp.sourceName, err)
			}
			continue
		}

		// Handle query arrays (slices)
		if fp.isSlice && fp.sourceType == "query" {
			values := r.URL.Query()[fp.sourceName]
			if len(values) > 0 {
				if err := f.setSliceField(fieldValue, values, fp.setter); err != nil {
					return fmt.Errorf("%s '%s': %w", fp.sourceType, fp.sourceName, err)
				}
			}
			continue
		}

		var value string
		var found bool

		// Get value based on source type
		switch fp.sourceType {
		case "header":
			value = r.Header.Get(fp.sourceName)
			found = value != ""
		case "route":
			value = r.PathValue(fp.sourceName)
			found = value != ""
		case "query":
			value = r.URL.Query().Get(fp.sourceName)
			found = value != ""
		}

		// Set field value using pre-computed setter (no type switch needed!)
		if found && value != "" {
			if err := fp.setter(fieldValue, value); err != nil {
				return fmt.Errorf("%s '%s': %w", fp.sourceType, fp.sourceName, err)
			}
		}
	}

	// Handle body field if present
	if parser.hasBodyField {
		bodyField := reqValue.Field(parser.bodyFieldIdx)
		if err := f.parseBody(r, bodyField); err != nil {
			return fmt.Errorf("body: %w", err)
		}
	}

	// Validate the entire request struct
	if err := f.validator.Struct(reqValue.Interface()); err != nil {
		validationErrors := f.formatValidationError(err, parser)
		if validationErrors != nil {
			return &validationErrorWrapper{validationErrors: validationErrors}
		}
		return err
	}

	return nil
}

// parseBody parses the request body
func (f *Framework) parseBody(r *http.Request, fieldValue reflect.Value) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	// Ensure the field is settable
	if !fieldValue.CanSet() {
		return fmt.Errorf("body field is not settable")
	}

	// Create a new instance of the field type
	newValue := reflect.New(fieldValue.Type())

	// Decode JSON body into the new instance
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(newValue.Interface()); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Set the field to the decoded value
	fieldValue.Set(newValue.Elem())

	return nil
}

// parseFileField parses a file upload from multipart form data
func (f *Framework) parseFileField(r *http.Request, fieldValue reflect.Value, formName string) error {
	// Parse multipart form (32MB max memory)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return fmt.Errorf("failed to parse multipart form: %w", err)
	}

	file, header, err := r.FormFile(formName)
	if err != nil {
		if err == http.ErrMissingFile {
			// File is optional if not required by validation
			return nil
		}
		return fmt.Errorf("failed to get form file: %w", err)
	}

	// Don't close the file here - let the handler do it
	// This allows the handler to read the file content

	// Create FileField value
	fileField := FileField{
		Filename: header.Filename,
		Size:     header.Size,
		Header:   header,
		Content:  file,
	}

	// Set the field value
	fieldValue.Set(reflect.ValueOf(fileField))

	return nil
}

// setSliceField sets a slice field from multiple string values
func (f *Framework) setSliceField(fieldValue reflect.Value, values []string, setter func(reflect.Value, string) error) error {
	// Create a new slice of the appropriate type
	sliceType := fieldValue.Type()

	newSlice := reflect.MakeSlice(sliceType, len(values), len(values))

	// Parse each value and add to slice
	for i, val := range values {
		elemValue := newSlice.Index(i)
		if err := setter(elemValue, val); err != nil {
			return fmt.Errorf("failed to parse element %d: %w", i, err)
		}
	}

	// Set the field to the new slice
	fieldValue.Set(newSlice)

	return nil
}

// formatValidationError formats validator errors into a structured format
func (f *Framework) formatValidationError(err error, parser *requestParser) []ValidationError {
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		// Build a map from struct path to tag info
		// For nested structs, the key will be like "Route.UserID" or "Body.Name"
		fieldTagMap := make(map[string]struct {
			tagName    string
			sourceType string
		})

		// Map field parsers to their struct field paths
		for _, fp := range parser.fieldParsers {
			parentFieldName := parser.requestType.Field(fp.fieldIndex).Name
			if fp.isNested {
				// Get the nested struct type
				parentType := parser.requestType.Field(fp.fieldIndex).Type
				nestedFieldName := parentType.Field(fp.nestedFieldIndex).Name
				structPath := parentFieldName + "." + nestedFieldName
				fieldTagMap[structPath] = struct {
					tagName    string
					sourceType string
				}{
					tagName:    fp.sourceName,
					sourceType: fp.sourceType,
				}
			}
		}

		// Handle body field separately
		if parser.hasBodyField {
			bodyFieldName := parser.requestType.Field(parser.bodyFieldIdx).Name
			fieldTagMap[bodyFieldName] = struct {
				tagName    string
				sourceType string
			}{
				tagName:    bodyFieldName,
				sourceType: "body",
			}
		}

		// Group errors by field with their tag info
		type fieldKey struct {
			name       string
			sourceType string
		}
		fieldErrorMap := make(map[fieldKey][]string)

		for _, e := range validationErrs {
			errorMsg := fmt.Sprintf("failed validation: %s", e.Tag())

			// Parse the namespace to determine the field path
			namespace := e.StructNamespace()
			parts := splitFieldPath(namespace)

			var actualFieldName string
			var sourceType string

			// Format: "RequestName.ParentField.NestedField" (3 parts) = nested field
			// Format: "RequestName.Body.FieldName" (3+ parts) = body field
			if len(parts) >= 3 {
				parentFieldName := parts[1]
				nestedFieldName := parts[2]
				structPath := parentFieldName + "." + nestedFieldName

				// Check if this is a Body field
				if parser.hasBodyField {
					bodyFieldName := parser.requestType.Field(parser.bodyFieldIdx).Name
					if parentFieldName == bodyFieldName {
						// This is a nested field in the body
						actualFieldName = e.Field()
						sourceType = "body"
					} else if tagInfo, ok := fieldTagMap[structPath]; ok {
						// This is a nested field in Route/Header/Query/Form
						actualFieldName = tagInfo.tagName
						sourceType = tagInfo.sourceType
					} else {
						actualFieldName = e.Field()
						sourceType = ""
					}
				} else if tagInfo, ok := fieldTagMap[structPath]; ok {
					// This is a nested field in Route/Header/Query/Form
					actualFieldName = tagInfo.tagName
					sourceType = tagInfo.sourceType
				} else {
					actualFieldName = e.Field()
					sourceType = ""
				}
			} else {
				// Top-level field (shouldn't happen with new system, but keep for safety)
				actualFieldName = e.Field()
				sourceType = ""
			}

			key := fieldKey{name: actualFieldName, sourceType: sourceType}
			fieldErrorMap[key] = append(fieldErrorMap[key], errorMsg)
		}

		// Convert map to slice of ValidationError structs
		validationErrors := make([]ValidationError, 0, len(fieldErrorMap))
		for key, errors := range fieldErrorMap {
			validationErrors = append(validationErrors, ValidationError{
				Field:      key.name,
				SourceType: key.sourceType,
				Errors:     errors,
			})
		}
		return validationErrors
	}
	return nil
}

// splitFieldPath splits a field namespace path (e.g., "CreateUserRequest.Body.Name")
func splitFieldPath(namespace string) []string {
	parts := []string{}
	current := ""
	for _, char := range namespace {
		if char == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// writeError writes an error response
func (f *Framework) writeError(w http.ResponseWriter, statusCode int, message string, details map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   message,
		Details: details,
	})
}

// writeValidationError writes a validation error response
func (f *Framework) writeValidationError(w http.ResponseWriter, statusCode int, validationErrors []ValidationError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ValidationErrorResponse{
		Error:  "validation failed",
		Fields: validationErrors,
	})
}

// ServeHTTP implements http.Handler
func (f *Framework) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mux.ServeHTTP(w, r)
}

// GetEndpoints returns all registered endpoints
func (f *Framework) GetEndpoints() []*EndpointSpec {
	return f.endpoints
}
