// Package inventorytest provides a shared test harness for validating the
// Installer, Uninstaller, and Lister contracts defined by internal/inventory,
// based on the behavior specified in the doc comments.
//
// Each distribution/<target> implementation can import this harness and run
// the contract tests against itself to verify that it satisfies the contracts
// declared by the inventory package.
//
// This package is test-only and is not intended to be imported from
// production code.
package inventorytest
