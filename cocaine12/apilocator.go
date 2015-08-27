package cocaine12

// ToDo: this file should be generated from c++

func newLocatorServiceInfo() *ServiceInfo {
	return &ServiceInfo{
		Endpoints: nil,
		Version:   1,
		API: dispatchMap{
			0: dispatchItem{
				Name:       "resolve",
				Downstream: EmptyDescription,
				Upstream: &streamDescription{
					0: &StreamDescriptionItem{
						Name:              "value",
						streamDescription: EmptyDescription,
					},
					1: &StreamDescriptionItem{
						Name:              "error",
						streamDescription: EmptyDescription,
					},
				},
			},
			1: dispatchItem{
				Name:       "connect",
				Downstream: EmptyDescription,
				Upstream: &streamDescription{
					0: &StreamDescriptionItem{
						Name:              "write",
						streamDescription: RecursiveDescription,
					},
					1: &StreamDescriptionItem{
						Name:              "error",
						streamDescription: EmptyDescription,
					},
					2: &StreamDescriptionItem{
						Name:              "close",
						streamDescription: EmptyDescription,
					},
				},
			},
			2: dispatchItem{
				Name:       "refresh",
				Downstream: EmptyDescription,
				Upstream: &streamDescription{
					0: &StreamDescriptionItem{
						Name:              "value",
						streamDescription: EmptyDescription,
					},
					1: &StreamDescriptionItem{
						Name:              "error",
						streamDescription: EmptyDescription,
					},
				},
			},
			3: dispatchItem{
				Name:       "cluster",
				Downstream: EmptyDescription,
				Upstream: &streamDescription{
					0: &StreamDescriptionItem{
						Name:              "value",
						streamDescription: EmptyDescription,
					},
					1: &StreamDescriptionItem{
						Name:              "error",
						streamDescription: EmptyDescription,
					},
				},
			},
		},
	}
}
