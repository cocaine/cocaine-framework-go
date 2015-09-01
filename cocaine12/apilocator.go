package cocaine12

var (
	EmptyDescription     = &streamDescription{}
	RecursiveDescription *streamDescription
)

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
						Name:        "value",
						Description: EmptyDescription,
					},
					1: &StreamDescriptionItem{
						Name:        "error",
						Description: EmptyDescription,
					},
				},
			},
			1: dispatchItem{
				Name:       "connect",
				Downstream: EmptyDescription,
				Upstream: &streamDescription{
					0: &StreamDescriptionItem{
						Name:        "write",
						Description: RecursiveDescription,
					},
					1: &StreamDescriptionItem{
						Name:        "error",
						Description: EmptyDescription,
					},
					2: &StreamDescriptionItem{
						Name:        "close",
						Description: EmptyDescription,
					},
				},
			},
			2: dispatchItem{
				Name:       "refresh",
				Downstream: EmptyDescription,
				Upstream: &streamDescription{
					0: &StreamDescriptionItem{
						Name:        "value",
						Description: EmptyDescription,
					},
					1: &StreamDescriptionItem{
						Name:        "error",
						Description: EmptyDescription,
					},
				},
			},
			3: dispatchItem{
				Name:       "cluster",
				Downstream: EmptyDescription,
				Upstream: &streamDescription{
					0: &StreamDescriptionItem{
						Name:        "value",
						Description: EmptyDescription,
					},
					1: &StreamDescriptionItem{
						Name:        "error",
						Description: EmptyDescription,
					},
				},
			},
		},
	}
}
