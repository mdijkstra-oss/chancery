package tokens

const charactersPerToken = 4

type Content interface {
	~string | ~[]byte
}

func Estimate[T Content](input []T) int {
	total := 0
	for _, value := range input {
		total += len(value)
	}
	return EstimateByteCount(total)
}

func EstimateByteCount(count int) int {
	return count / charactersPerToken
}
