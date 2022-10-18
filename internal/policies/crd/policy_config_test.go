package crd

// func TestGetPolicyConfig(t *testing.T) {
// 	cases := []struct {
// 		entity  domain.Entity
// 		configs pacv2.PolicyConfigList
// 		result  domain.PolicyConfig
// 	}{
// 		{
// 			entity: domain.Entity{
// 				Name:      "deployment-1",
// 				Namespace: "default",
// 			},
// 			result: domain.PolicyConfig{
// 				Config: map[string]domain.PolicyConfigConfig{
// 					"policy-1": domain.PolicyConfigConfig{
// 						Parameters: map[string]domain.PolicyConfigParameter{
// 							"param-1": domain.PolicyConfigParameter{
// 								Value:     "default",
// 								ConfigRef: "config-1",
// 							},
// 						},
// 					},
// 				},
// 			},
// 			configs: pacv2.PolicyConfigList{
// 				Items: []pacv2.PolicyConfig{
// 					{
// 						TypeMeta: v1.TypeMeta{
// 							APIVersion: pacv2.GroupVersion.Version,
// 							Kind:       pacv2.PolicyConfigKind,
// 						},
// 						ObjectMeta: v1.ObjectMeta{
// 							Name:      "config-1",
// 							Namespace: "default",
// 						},
// 						Spec: pacv2.PolicyConfigSpec{
// 							Config: map[string]pacv2.PolicyConfigConfig{
// 								"policy-1": pacv2.PolicyConfigConfig{
// 									Parameters: map[string]apiextensionsv1.JSON{
// 										"param-1": apiextensionsv1.JSON{[]byte("default")},
// 									},
// 								},
// 							},
// 						},
// 					},
// 					{
// 						TypeMeta: v1.TypeMeta{
// 							APIVersion: pacv2.GroupVersion.Version,
// 							Kind:       pacv2.PolicyConfigKind,
// 						},
// 						ObjectMeta: v1.ObjectMeta{
// 							Name:      "config-2",
// 							Namespace: "flux-system",
// 						},
// 						Spec: pacv2.PolicyConfigSpec{
// 							Config: map[string]pacv2.PolicyConfigConfig{
// 								"policy-1": pacv2.PolicyConfigConfig{
// 									Parameters: map[string]apiextensionsv1.JSON{
// 										"param-1": apiextensionsv1.JSON{[]byte("flux")},
// 									},
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// fake.
// watcher, err := NewPoliciesWatcher(context.Background(), , Config{})

// for i := range cases {

// }
// }
