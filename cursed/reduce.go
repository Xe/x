package cursed

import "golang.org/x/exp/constraints"

func Reduce[D any](data []D, doer func(D, D) D) D {
	var initial D

	return Fold[D, D](initial, data, doer)
}

func Fold[T any, U any](initial U, data []T, doer func(T, U) U) U {
	acc := initial
	for _, d := range data {
		temp := doer(d, acc)
		acc = temp
	}

	return acc
}

type Number interface {
	constraints.Float | constraints.Integer
}

func Sum[T Number](data []T) T {
	return Reduce[T](data, func(x, y T) T {
		return x + y
	})
}

func Max[T Number](data []T) T {
	return Reduce[T](data, func(x, y T) T {
		if x > y {
			return x
		} else if y < x {
			return y
		} else {
			return x // x == y
		}
	})
}

func Min[T Number](data []T) T {
	return Reduce[T](data, func(x, y T) T {
		if x > y {
			return y
		} else if y < x {
			return x
		} else {
			return x // x == y
		}
	})
}
