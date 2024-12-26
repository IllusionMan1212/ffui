package main

func contains[T comparable](slice []T, elem T) bool {
	for _, e := range slice {
		if e == elem {
			return true
		}
	}

	return false
}

func every[T any](slice []T, pred func(elem T) bool) bool {
	for _, e := range slice {
		if !pred(e) {
			return false
		}
	}

	return true
}

func anyOf[T any](slice []T, pred func(elem T) bool) bool {
	for _, e := range slice {
		if pred(e) {
			return true
		}
	}

	return false
}

func filter[T any](slice []T, pred func(elem T) bool) []T {
	filtered := make([]T, 0)

	for _, e := range slice {
		if pred(e) {
			filtered = append(filtered, e)
		}
	}

	return filtered
}
