package pagination

func Collect[T any](fn func(cursor string) (resp T, nextCursor string, err error)) ([]T, error) {
	return Reduce(fn, func(resp T, acc []T) ([]T, error) {
		acc = append(acc, resp)
		return acc, nil
	})
}

func Reduce[T any, R any](
	fetch func(cursor string) (resp T, nextCursor string, err error),
	combine func(resp T, acc R) (R, error),
) (R, error) {
	cursor := ""
	var r R

	for {
		resp, nextCursor, err := fetch(cursor)
		if err != nil {
			return r, err
		}

		r, err = combine(resp, r)
		if err != nil {
			return r, err
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}
	return r, nil

}
