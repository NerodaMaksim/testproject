package types

type Element[K comparable, V any] struct {
	next *Element[K, V]
	prev *Element[K, V]

	Key   K
	Value V
}

func (e *Element[K, V]) Next() *Element[K, V] {
	return e.next
}

func (e *Element[K, V]) Prev() *Element[K, V] {
	return e.prev
}

type List[K comparable, V any] struct {
	root Element[K, V] // list head and tail
}

func (l *List[K, V]) IsEmpty() bool {
	return l.root.next == nil
}

func (l *List[K, V]) First() *Element[K, V] {
	return l.root.next
}

func (l *List[K, V]) Last() *Element[K, V] {
	return l.root.prev
}

func (l *List[K, V]) Remove(e *Element[K, V]) {
	if e.prev == nil {
		l.root.next = e.next
	} else {
		e.prev.next = e.next
	}

	if e.next == nil {
		l.root.prev = e.prev
	} else {
		e.next.prev = e.prev
	}

	e.next = nil
	e.prev = nil
}

func (l *List[K, V]) Push(key K, value V) *Element[K, V] {
	e := &Element[K, V]{Key: key, Value: value}

	if l.root.prev == nil {
		// If first element
		l.root.next = e
		l.root.prev = e

		return e
	}

	e.prev = l.root.prev
	l.root.prev.next = e
	l.root.prev = e

	return e
}
