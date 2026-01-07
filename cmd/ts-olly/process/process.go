package process

type Process struct {
	name      string
	instances []*instance
}

func (p Process) Name() string {
	return p.name
}

func (p Process) Instances() []*instance {
	return p.instances
}
