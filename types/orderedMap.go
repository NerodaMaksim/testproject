package types

type OrderedMap[K comparable, V any] struct {
	kv map[K]*Element[K, V]
	ll List[K, V]
}

func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{
		kv: make(map[K]*Element[K, V]),
	}
}

func (m *OrderedMap[K, V]) Get(key K) (value V, ok bool) {
	v, ok := m.kv[key]
	if ok {
		value = v.Value
	}

	return
}

func (m *OrderedMap[K, V]) Set(key K, value V) bool {
	_, alreadyExist := m.kv[key]
	if alreadyExist {
		m.kv[key].Value = value
		return false
	}

	element := m.ll.Push(key, value)
	m.kv[key] = element
	return true
}

func (m *OrderedMap[K, V]) Len() int {
	return len(m.kv)
}

func (m *OrderedMap[K, V]) Keys() (keys []K) {
	keys = make([]K, 0, m.Len())
	for el := m.ll.First(); el != nil; el = el.Next() {
		keys = append(keys, el.Key)
	}

	return
}

func (m *OrderedMap[K, V]) Delete(key K) (didDelete bool) {
	element, ok := m.kv[key]
	if ok {
		m.ll.Remove(element)
		delete(m.kv, key)
	}

	return ok
}
