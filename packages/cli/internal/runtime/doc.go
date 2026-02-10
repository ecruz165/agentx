// Package runtime defines the Runtime interface for executing skills and provides
// implementations for the Node.js and Go runtimes. The DispatchRuntime function
// selects the correct runtime based on the skill manifest's runtime field.
package runtime