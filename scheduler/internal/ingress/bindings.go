package ingress

import v1 "code.cloudfoundry.org/scalable-syslog/internal/api/v1"

type Bindings []v1.Binding

func (b Bindings) DrainCount(search v1.Binding) int {
	count := 0
	for _, binding := range b {
		if binding == search {
			count++
		}
	}
	return count
}
