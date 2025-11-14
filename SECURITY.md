# Crystal Logging Framework - Security Features

## Overview
This document describes the security features implemented in the Crystal logging framework to protect against common vulnerabilities and ensure safe logging practices.

## Log Injection Prevention

Crystal prevents log injection attacks by sanitizing message content before formatting. The text formatter automatically escapes newline characters and carriage returns to prevent attackers from injecting fake log entries.

### How It Works
- Newlines (`\n`) are replaced with `\\n`
- Carriage returns (`\r`) are replaced with `\\r`
- This prevents attackers from creating fake log entries that could be used to hide malicious activity or confuse log analysis systems

### Example
```go
// Vulnerable code (without protection):
logger.Info("User login\nERROR: Failed to validate credentials")

// With Crystal's protection, this becomes:
// INFO User login\\nERROR: Failed to validate credentials
```

## Sensitive Data Masking

Crystal includes built-in sensitive data masking to prevent accidental logging of credentials, tokens, and other confidential information.

### Configuration
```go
formatter := &core.TextFormatter{
    MaskSensitiveData: true,
    MaskString:        "***",
}
```

### Automatic Masking
The formatter automatically masks fields with keys that indicate sensitive data:
- Fields containing "password"
- Fields containing "token"
- Fields containing "secret"
- Fields containing "key"

### Example
```go
// Without masking:
logger.Info("User login", "username", "john_doe", "password", "secret123")
// Output: INFO User login {username="john_doe" password="secret123"}

// With masking enabled:
logger.Info("User login", "username", "john_doe", "password", "secret123")
// Output: INFO User login {username="john_doe" password="***"}
```

## Secure Defaults

Crystal is configured with secure defaults to minimize the risk of accidental exposure:

1. **No sensitive data masking by default** - Opt-in feature to prevent performance impact
2. **No color output by default** - Prevents injection through ANSI escape sequences
3. **Limited field width** - Prevents excessive memory consumption
4. **No stack traces by default** - Prevents information disclosure

## Best Practices

1. **Enable sensitive data masking** in production environments
2. **Review log output** regularly for potential security issues
3. **Use structured logging** to ensure proper field handling
4. **Limit log retention** to reduce exposure window
5. **Monitor log access** to detect unauthorized access

## Testing

Crystal includes comprehensive security tests to verify protection mechanisms:

- Log injection prevention tests
- Sensitive data masking tests
- Input validation tests (planned)
- Secure defaults verification (planned)

These tests ensure that security features continue to work as expected across updates.