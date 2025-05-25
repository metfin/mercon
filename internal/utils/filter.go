package utils

// Filter applies a filter function to each element in a slice
// and returns a new slice containing only the elements for which the filter function returns true.
func Filter[T any](slice []T, filterFunc func(T) bool) []T {
	var result []T
	for _, item := range slice {
		if filterFunc(item) {
			result = append(result, item)
		}
	}
	return result
}
