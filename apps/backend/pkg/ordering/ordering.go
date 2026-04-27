package ordering

type Direction int

const (
	DirectionAsc Direction = iota
	DirectionDesc
)

type Ordering[T comparable] struct {
	Field     T
	Direction Direction
}

type Orderings[T comparable] []Ordering[T]

func OrderByAsc[T comparable](field T) Ordering[T] {
	return Ordering[T]{
		Field:     field,
		Direction: DirectionAsc,
	}
}

func OrderByDesc[T comparable](field T) Ordering[T] {
	return Ordering[T]{
		Field:     field,
		Direction: DirectionDesc,
	}
}
