# Input Task

**Task ID:** TASK-GOLDEN-002  
**Tier:** standard  
**Owner:** alice

Refactor the authentication module to support OAuth2 PKCE flow.

## Requirements

- Implement PKCE code challenge and verifier.
- Update login redirect to include `code_challenge`.
- Add tests covering success and error paths.

## Evidence needed

- Changed files list.
- Test output.
- Security review note.
