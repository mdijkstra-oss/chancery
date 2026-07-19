package fn

func Map[A, B any](items []A, transform func(A) B) []B {
	result := make([]B, len(items))
	for index, item := range items {
		result[index] = transform(item)
	}
	return result
}

func Filter[T any](items []T, keep func(T) bool) []T {
	result := make([]T, 0, len(items))
	for _, item := range items {
		if keep(item) {
			result = append(result, item)
		}
	}
	return result
}
