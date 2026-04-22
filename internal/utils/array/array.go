package array

func Difference[T comparable](a, b []T) (diff []T) {
	m := make(map[T]bool)

	for _, item := range b {
		m[item] = true
	}

	for _, item := range a {
		if _, ok := m[item]; !ok {
			diff = append(diff, item)
		}
	}

	return
}

func Chunk[T any](input []T, chunkSize int) (chunks [][]T) {
	for i := 0; i < len(input); i += chunkSize {
		end := min(i+chunkSize, len(input))
		chunks = append(chunks, input[i:end])
	}
	return chunks
}

func Filter[T any](input []T, testFunc func(T) bool) []T {
	resp := make([]T, 0, len(input))
	for _, item := range input {
		if testFunc(item) {
			resp = append(resp, item)
		}
	}
	return resp
}

func Find[T any](input []T, testFunc func(T) bool) (item T, ok bool) {
	for _, item := range input {
		if testFunc(item) {
			return item, true
		}
	}

	var zero T
	return zero, false
}

func Map[T any, R any](input []T, mapFunc func(T) R) []R {
	v := make([]R, len(input))
	for i, item := range input {
		v[i] = mapFunc(item)
	}
	return v
}

func Unique[T any](slice []T, property func(T) uint) []T {
	unique := make(map[uint]T)

	for _, item := range slice {
		id := property(item)
		if _, exists := unique[id]; !exists {
			unique[id] = item
		}
	}

	result := make([]T, 0, len(unique))
	for _, item := range unique {
		result = append(result, item)
	}

	return result
}
