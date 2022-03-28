package testdata

var (
	ValidadmissionBody = []byte(`
	{
		"uid": "705ab4f5-6393-11e8-b7cc-42010a800002",
	
		"kind": {"group":"apps","version":"v1","kind":"Deployment"},
		"resource": {"group":"apps","version":"v1","resource":"deployments"},
		"subResource": "scale",
		"requestKind": {"group":"apps","version":"v1","kind":"Deployment"},
		"requestResource": {"group":"apps","version":"v1","resource":"deployments"},
		"requestSubResource": "scale",
	
		"name": "nginx-deployment",
		"namespace": "unit-testing",
	
		"operation": "CREATE",
	
		"userInfo": {
		"username": "admin",
		"uid": "014fbff9a07c",
		"groups": ["system:authenticated","my-admin-group"]
		},
	
		"object": {
		"apiVersion": "apps/v1",
		"kind": "Deployment",
		"metadata": {
			"name": "nginx-deployment",
			"labels": {
			"app": "nginx"
			},
			"namespace": "unit-testing"
		},
		"spec": {
			"replicas": 3,
			"selector": {
			"matchLabels": {
				"app": "nginx"
			}
			},
			"template": {
			"metadata": {
				"labels": {
				"app": "nginx"
				}
			},
			"spec": {
				"containers": [
				{
					"name": "nginx",
					"image": "nginx:latest",
					"ports": [
					{
						"containerPort": 80
					}
					]
				}
				]
			}
			}
		}
		},
		
		"dryRun": false
	  }
	`)
	InvalidadmissionEntity = []byte(`
	{
		"uid": "705ab4f5-6393-11e8-b7cc-42010a800002",
	
		"kind": {"group":"apps","version":"v1","kind":"Deployment"},
		"resource": {"group":"apps","version":"v1","resource":"deployments"},
		"subResource": "scale",
		"requestKind": {"group":"apps","version":"v1","kind":"Deployment"},
		"requestResource": {"group":"apps","version":"v1","resource":"deployments"},
		"requestSubResource": "scale",
	
		"name": "nginx-deployment",
		"namespace": "unit-testing",
	
		"operation": "CREATE",
	
		"userInfo": {
		"username": "admin",
		"uid": "014fbff9a07c",
		"groups": ["system:authenticated","my-admin-group"]
		},
	
		"object": "invalid entity",
		
		"dryRun": false
	  }
	`)
	SkippedadmissionBody = []byte(`
	{
		"uid": "705ab4f5-6393-11e8-b7cc-42010a800002",
	
		"kind": {"group":"apps","version":"v1","kind":"Deployment"},
		"resource": {"group":"apps","version":"v1","resource":"deployments"},
		"subResource": "scale",
		"requestKind": {"group":"apps","version":"v1","kind":"Deployment"},
		"requestResource": {"group":"apps","version":"v1","resource":"deployments"},
		"requestSubResource": "scale",
	
		"name": "nginx-deployment",
		"namespace": "kube-system",
	
		"operation": "CREATE",
	
		"userInfo": {
		"username": "admin",
		"uid": "014fbff9a07c",
		"groups": ["system:authenticated","my-admin-group"]
		},
	
		"object": {
		},
		
		"dryRun": false
	  }
	`)
)
