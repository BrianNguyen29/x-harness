# Input Task

**Task ID:** TASK-GOLDEN-004  
**Tier:** light  
**Owner:** alice

Add rate-limiting middleware to the API gateway.

## Requirements

- Limit requests to 100 per minute per IP.
- Return 429 status when limit exceeded.
- Add Redis-backed sliding window.

## Evidence needed

- Middleware implementation.
- Integration tests.
- Load test results.
