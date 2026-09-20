// Package version is what this binary calls itself.
//
// ⚠️ Its own constant rather than an import from the control plane: they are separate
// modules and separate deployments, and a restaurant's backend must not need ours to
// build. Kept equal by control/internal/handlers/version_test.go, which reads the root
// VERSION file and every declaration of it.
package version

const Version = "v0.2.1"
