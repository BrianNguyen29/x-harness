// Source file used by the boundary-violation golden fixture. The
// forbidden import below triggers the deny rule in
// boundary-policy.yaml when xh verify runs with
// --boundary-enforce block_high or block_all. Golden regression
// coverage is exercised by copying the fixture directory to a temp
// root and invoking verify from a Go test (see
// internal/cli/verify_boundary_test.go).
import { getUser } from "internal/db/users";

export function login(userId) {
  return getUser(userId);
}
