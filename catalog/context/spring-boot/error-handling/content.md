# Spring Boot Error Handling Patterns

## Global Exception Handling with @ControllerAdvice

Use `@ControllerAdvice` to centralize exception handling across all controllers:

```java
@ControllerAdvice
public class GlobalExceptionHandler extends ResponseEntityExceptionHandler {

    @ExceptionHandler(ResourceNotFoundException.class)
    public ResponseEntity<ErrorResponse> handleNotFound(ResourceNotFoundException ex) {
        ErrorResponse error = new ErrorResponse(
            HttpStatus.NOT_FOUND.value(),
            ex.getMessage(),
            Instant.now()
        );
        return ResponseEntity.status(HttpStatus.NOT_FOUND).body(error);
    }

    @ExceptionHandler(ConstraintViolationException.class)
    public ResponseEntity<ErrorResponse> handleValidation(ConstraintViolationException ex) {
        ErrorResponse error = new ErrorResponse(
            HttpStatus.BAD_REQUEST.value(),
            "Validation failed: " + ex.getMessage(),
            Instant.now()
        );
        return ResponseEntity.badRequest().body(error);
    }

    @ExceptionHandler(Exception.class)
    public ResponseEntity<ErrorResponse> handleGeneral(Exception ex) {
        ErrorResponse error = new ErrorResponse(
            HttpStatus.INTERNAL_SERVER_ERROR.value(),
            "An unexpected error occurred",
            Instant.now()
        );
        return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).body(error);
    }
}
```

## Standard Error Response Record

Use a Java record for consistent error responses:

```java
public record ErrorResponse(
    int status,
    String message,
    Instant timestamp
) {}
```

## Custom Exception Hierarchy

Define domain-specific exceptions extending `RuntimeException`:

```java
public class ResourceNotFoundException extends RuntimeException {
    public ResourceNotFoundException(String resource, Object id) {
        super("%s not found with id: %s".formatted(resource, id));
    }
}

public class BusinessRuleException extends RuntimeException {
    private final String code;

    public BusinessRuleException(String code, String message) {
        super(message);
        this.code = code;
    }

    public String getCode() { return code; }
}
```

## Best Practices

- Always return a consistent error response structure across all endpoints
- Log exceptions at the handler level, not in service code
- Never expose stack traces or internal details in production responses
- Use `@ResponseStatus` on exception classes only for simple cases
- Include correlation IDs in error responses for distributed tracing
- Map validation errors to specific field-level messages
