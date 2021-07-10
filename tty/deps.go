package tty

type dep struct {
	from int
	to   int
}

type depSet struct {
	deps []dep
}

func (ds *depSet) add(from, to int) {
	ds.deps = append(ds.deps, dep{from, to})
}

func (ds *depSet) blocked() map[int]bool {
	out := map[int]bool{}
	for _, item := range ds.deps {
		out[item.from] = true
	}
	return out
}

func (ds *depSet) unblock(to int) {
	var out []dep
	for _, item := range ds.deps {
		if item.to != to {
			out = append(out, item)
		}
	}
	ds.deps = out
}
