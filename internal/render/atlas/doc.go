// Package atlas manages GPU texture atlases for both fonts and images. It provides
// a static SDF font atlas for text rendering and dynamic image atlas allocation
// with shelf-packing and LRU eviction, including dirty region tracking for
// efficient GPU uploads.
package atlas
