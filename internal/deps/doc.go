// Package deps implements dependency declaration, resolution, probing, and installation
// for easyrice rice packages. It is a leaf package: it does NOT import internal/manifest.
// Import direction: manifest → deps (one-way only).
package deps
