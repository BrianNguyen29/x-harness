# Input Task

**Task ID:** TASK-GOLDEN-003  
**Tier:** light  
**Owner:** alice

Fix the off-by-one error in the pagination helper.

## Requirements

- `page` parameter must be 1-based.
- Offset calculation must use `(page - 1) * limit`.
- Existing tests must pass.
