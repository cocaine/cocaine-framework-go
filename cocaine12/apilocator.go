package cocaine12

var (
	emptyDescription     = &streamDescription{}
	recursiveDescription *streamDescription
)

func newLocatorServiceInfo() *ServiceInfo {
	return &ServiceInfo{
		Endpoints: nil,
		Version:   1,
		API: dispatchMap{
			0: dispatchItem{
				Name:       "resolve",
				Downstream: emptyDescription,
				Upstream: &streamDescription{
					0: &StreamDescriptionItem{
						Name:        "value",
						Description: emptyDescription,
					},
					1: &StreamDescriptionItem{
						Name:        "error",
						Description: emptyDescription,
					},
				},
			},
			1: dispatchItem{
				Name:       "connect",
				Downstream: emptyDescription,
				Upstream: &streamDescription{
					0: &StreamDescriptionItem{
						Name:        "write",
						Description: recursiveDescription,
					},
					1: &StreamDescriptionItem{
						Name:        "error",
						Description: emptyDescription,
					},
					2: &StreamDescriptionItem{
						Name:        "close",
						Description: emptyDescription,
					},
				},
			},
			2: dispatchItem{
				Name:       "refresh",
				Downstream: emptyDescription,
				Upstream: &streamDescription{
					0: &StreamDescriptionItem{
						Name:        "value",
						Description: emptyDescription,
					},
					1: &StreamDescriptionItem{
						Name:        "error",
						Description: emptyDescription,
					},
				},
			},
			3: dispatchItem{
				Name:       "cluster",
				Downstream: emptyDescription,
				Upstream: &streamDescription{
					0: &StreamDescriptionItem{
						Name:        "value",
						Description: emptyDescription,
					},
					1: &StreamDescriptionItem{
						Name:        "error",
						Description: emptyDescription,
					},
				},
			},
		},
	}
}
