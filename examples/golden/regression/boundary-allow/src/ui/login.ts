// Source file used by the boundary-allow golden fixture. The import
// path below falls under the `internal/db/public/**` allow list in
// boundary-policy.yaml, so no boundary violation is raised under any
// enforcement mode. Golden regression coverage is exercised by
// copying the fixture directory to a temp root and invoking verify
// from a Go test (see internal/cli/verify_boundary_test.go).
import { getUser } from "internal/db/public/users";

export function login(userId) {
  return getUser(userId);
}
