package maps

func Keys(m map[string]interface{}) (keys []string) {
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func Merge(dst map[string]interface{}, src map[string]interface{}, mergeSlices bool) {
	if len(src) == 0 {
		return
	}
	Walk(dst, src, VisitFunc(func(info *NodeInfo) error {
		if info.Recurse || !info.Exists2 {
			return nil
		}
		if mergeSlices {
			// todo: handle more types of slices. not sure whether to fall back on reflection though
			switch s1 := info.V1.(type) {
			case []interface{}:
				if s2, ok2 := info.V2.([]interface{}); ok2 {
					v2:
					for _, v2 := range s2 {
						for _, v1 := range s1 {
							if v1 == v2 {
								continue v2
							}
						}
						s1 = append(s1, v2)
					}
					info.M1[info.Key] = s1
					return nil
				}
			}
		}
		info.M1[info.Key] = info.V2
		return nil
	}))
}

func Contains(m1, m2 map[string]interface{}) bool {
	if len(m2) > len(m1) {
		return false
	}
	result := true
	Walk(m1, m2, VisitFunc(func(info *NodeInfo) error {
		// if this node is recursable, recurse immediately
		if info.Recurse {
			return nil
		}
		// if m1 doesn't have the key at all, m1 doesn't contain m2
		if !info.Exists1 {
			result = false
			info.Continue = false
			return nil
		}
		// if the values are both slices, check if slice 1 contains slice 2

		return nil
	}))
	return result
}

type NodeInfo struct {
	Key               string
	M1, M2            map[string]interface{}
	V1, V2            interface{}
	Exists1, Exists2  bool
	Recurse, Continue bool
}

func Walk(m1, m2 map[string]interface{}, v Visitor) error {
	// track which keys we've visited.  We are modifying the map
	// we're iterating over, which can cause us to see the same
	// key more than once.
	visited := map[string]bool{}
	info := &NodeInfo{M1: m1, M2: m2, Continue:true}

	for info.Key, info.V1 = range m1 {
		if visited[info.Key] {
			continue
		}
		visited[info.Key] = true
		info.Exists1 = true
		info.V2, info.Exists2 = m2[info.Key]
		c1, isMap1 := info.V1.(map[string]interface{})
		c2, isMap2 := info.V2.(map[string]interface{})
		info.Recurse = isMap1 && isMap2
		if err := v.Visit(info); err != nil {
			return err
		}
		if !info.Continue {
			return nil
		}
		if info.Recurse {
			Walk(c1, c2, v)
		}
	}
	for info.Key, info.V2 = range m2 {
		if visited[info.Key] {
			continue
		}
		visited[info.Key] = true
		// don't need to check for recursion here, because these keys
		// we're only in m2
		info.Exists1 = false
		info.Exists2 = true
		info.Recurse = false
		info.V1 = nil
		if err := v.Visit(info); err != nil {
			return err
		}
		if !info.Continue {
			return nil
		}
	}
	return nil
}

type Visitor interface {
	Visit(info *NodeInfo) error
}

type VisitFunc func(info *NodeInfo) error

func (f VisitFunc) Visit(info *NodeInfo) error {
	return f(info)
}


