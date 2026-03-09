# PixelForge - Error Log

## Build Errors

### Error #1: Content-Type Detection in Upload
**Error:** Tests failed with `unsupported image type: application/octet-stream` — multipart form file header returned `application/octet-stream` for PNG data.
**Cause:** `header.Header.Get("Content-Type")` relies on the multipart header which may not accurately reflect the actual file type, especially in tests.
**Fix:** Changed to detect content type from the actual file bytes using `http.DetectContentType(data)` instead of trusting the multipart header.

---

## Security Audit (10-Point Checklist)

| # | Category | Result | Notes |
|---|----------|--------|-------|
| 1 | Hardcoded Secrets | PASS | No secrets in code. Port and storage dir from env vars |
| 2 | SQL Injection | N/A | No database — in-memory job storage, file-based image storage |
| 3 | Input Validation | PASS | File size limit (10MB), content type validation, dimension limits (10000x10000), angle validation |
| 4 | Dependency Vulnerabilities | PASS | Only stdlib dependencies — no third-party packages |
| 5 | Auth / Access Control | NOTE | No authentication — acceptable for demo/portfolio. Production would need API keys |
| 6 | CSRF / XSS / Security Headers | PASS | All security headers set: CSP, HSTS, X-Frame-Options, X-XSS-Protection, X-Content-Type-Options, Permissions-Policy, Referrer-Policy |
| 7 | Sensitive Data Exposure | PASS | No stack traces returned. Structured JSON error responses only |
| 8 | Docker Security | PASS | Multi-stage build, alpine base, non-root user (appuser), healthcheck |
| 9 | CI Security | PASS | Pinned action versions, go vet included, race detector enabled in tests |
| 10 | Rate Limiting / DoS | PASS | Rate limiter middleware (100 req/min), MaxBytesReader on request body, server timeouts (read: 15s, write: 30s, idle: 60s), graceful shutdown |

### Security Features Implemented
1. Rate limiting middleware (100 requests/minute per IP)
2. Request body size limit (10MB via MaxBytesReader)
3. File type validation via content sniffing (not header trust)
4. Image dimension limits (max 10000x10000 pixels)
5. Security headers on all responses
6. Graceful shutdown with 10-second timeout
7. Server timeouts (read/write/idle)
8. File permissions: 0750 for dirs, 0600 for files
9. Path traversal prevention via filepath.Base()
10. No third-party dependencies — stdlib only
