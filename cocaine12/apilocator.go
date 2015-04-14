package cocaine12

// ToDo: this file should be generated from c++

func newLocatorServiceInfo() *ServiceInfo {
	return &ServiceInfo{
		Endpoints: nil,
		Version:   1,
		API: DispatchMap{
			0: DispatchItem{
				Name:       "resolve",
				Downstream: EmptyDescription,
				Upstream: &StreamDescription{
					0: &StreamDescriptionItem{
						Name:              "value",
						StreamDescription: EmptyDescription,
					},
					1: &StreamDescriptionItem{
						Name:              "error",
						StreamDescription: EmptyDescription,
					},
				},
			},
			1: DispatchItem{
				Name:       "connect",
				Downstream: EmptyDescription,
				Upstream: &StreamDescription{
					0: &StreamDescriptionItem{
						Name:              "write",
						StreamDescription: RecursiveDescription,
					},
					1: &StreamDescriptionItem{
						Name:              "error",
						StreamDescription: EmptyDescription,
					},
					2: &StreamDescriptionItem{
						Name:              "close",
						StreamDescription: EmptyDescription,
					},
				},
			},
			2: DispatchItem{
				Name:       "refresh",
				Downstream: EmptyDescription,
				Upstream: &StreamDescription{
					0: &StreamDescriptionItem{
						Name:              "value",
						StreamDescription: EmptyDescription,
					},
					1: &StreamDescriptionItem{
						Name:              "error",
						StreamDescription: EmptyDescription,
					},
				},
			},
			3: DispatchItem{
				Name:       "cluster",
				Downstream: EmptyDescription,
				Upstream: &StreamDescription{
					0: &StreamDescriptionItem{
						Name:              "value",
						StreamDescription: EmptyDescription,
					},
					1: &StreamDescriptionItem{
						Name:              "error",
						StreamDescription: EmptyDescription,
					},
				},
			},
		},
	}
}
