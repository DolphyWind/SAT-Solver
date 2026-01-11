package main

func remove_at_index[T any](s []T, i int) []T {
    if i < 0 || i >= len(s) {
        return s
    }
    return append(s[:i], s[i+1:]...)
}
