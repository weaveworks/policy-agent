package testdata

import (
	"github.com/weaveworks/policy-agent/pkg/policy-core/domain"
	"github.com/weaveworks/policy-agent/pkg/uuid-go"
)

var (
	Entity = `
{
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
		  "securityContext": { "runAsNonRoot": false },
		  "containers": [
			{
			  "name": "nginx",
			  "image": "nginx:latest",
			  "ports": [
				{
				  "containerPort": 80
				}
			  ],
			  "securityContext": { "runAsNonRoot": false }
			}
		  ]
		}
	  }
	}
  }`
	CompliantEntity = `
{
	"apiVersion": "apps/v1",
	"kind": "Deployment",
	"metadata": {
	  "name": "nginx-deployment",
	  "labels": {
		"app": "nginx",
		"owner": "unit.test"
	  }
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
			  "image": "nginx:1.0.0",
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
  }`
	Policies = map[string]domain.Policy{
		"imageTag": {
			Name: "Using latest image tag in container",
			ID:   uuid.NewV4().String(),
			Code: `
		package magalix.advisor.images.image_tag_enforce

		image_tag := input.parameters.image_tag
		exclude_namespace := input.parameters.exclude_namespace
		exclude_label_key := input.parameters.exclude_label_key
		exclude_label_value := input.parameters.exclude_label_value

		violation[result] {
		not exclude_namespace == controller_input.metadata.namespace
		not exclude_label_value == controller_input.metadata.labels[exclude_label_key]
		some i
		containers = controller_spec.containers[i]
		splittedUrl = split(containers.image, "/")
		image = splittedUrl[count(splittedUrl)-1]
		not contains(image, ":")
		result = {
			"issue detected": true,
			"msg": "Image is not tagged",
			"violating_key": sprintf("spec.template.spec.containers[%v].image", [i])
		}
		}

		violation[result] {
		some i
		containers = controller_spec.containers[i]
		splittedUrl = split(containers.image, "/")
		image = splittedUrl[count(splittedUrl)-1]
		count(split(image, ":")) == 2
		[image_name, tag] = split(image, ":")
		tag == image_tag
		result = {
			"issue detected": true,
			"msg": sprintf("Image contains unapproved tag '%v'", [image_tag]),
			"image": image,
			"violating_key": sprintf("spec.template.spec.containers[%v].image", [i])
		}
		}

		violation[result] {
		some i
		containers = controller_spec.containers[i]
		splittedUrl = split(containers.image, "/")
		image = splittedUrl[count(splittedUrl)-1]
		count(split(image, ":")) == 3
		[image_name, port, tag] = split(image, ":")
		tag == image_tag
		result = {
			"issue detected": true,
			"msg": sprintf("Image contains unapproved tag:'%v'", [image_tag]),
			"image": image,
			"violating_key": sprintf("spec.template.spec.containers[%v].image", [i])
		}
		}

		# Controller input
		controller_input = input.review.object

		# controller_container acts as an iterator to get containers from the template
		controller_spec = controller_input.spec.template.spec {
		contains_kind(controller_input.kind, {"StatefulSet" , "DaemonSet", "Deployment", "Job"})
		} else = controller_input.spec {
		controller_input.kind == "Pod"
		} else = controller_input.spec.jobTemplate.spec.template.spec {
		controller_input.kind == "CronJob"
		}

		contains_kind(kind, kinds) {
		kinds[_] = kind
		}`,
			Mutate: true,
			Parameters: []domain.PolicyParameters{
				{
					Name:     "image_tag",
					Type:     "string",
					Required: true,
					Value:    "latest",
				},
				{
					Name:     "exclude_namespace",
					Type:     "string",
					Required: false,
					Value:    "",
				},
				{
					Name:     "exclude_label_key",
					Type:     "string",
					Required: false,
					Value:    "",
				},
				{
					Name:     "exclude_label_value",
					Type:     "string",
					Required: false,
					Value:    "",
				},
			},
		},
		"missingOwner": {
			Name: "Missing owner label in metadata",
			ID:   uuid.NewV4().String(),
			Code: `
			package magalix.advisor.labels.missing_owner_label

			exclude_namespace := input.parameters.exclude_namespace
			exclude_label_key := input.parameters.exclude_label_key
			exclude_label_value := input.parameters.exclude_label_value
		
			violation[result] {
			  not exclude_namespace == controller_input.metadata.namespace
			  not exclude_label_value == controller_input.metadata.labels[exclude_label_key]
			  # Filter the type of entity before moving on since this shouldn't apply to all entities
			  label := "owner"
			  contains_kind(controller_input.kind, {"StatefulSet" , "DaemonSet", "Deployment", "Job"})
			  not controller_input.metadata.labels[label]
			  result = {
				"issue detected": true,
				"msg": sprintf("you are missing a label with the key '%v'", [label]),
				"violating_key": "metadata.labels.owner",
				"recommended_value": "test"
			  }
			}
		
			# Controller input
			controller_input = input.review.object
		
			contains_kind(kind, kinds) {
			  kinds[_] = kind
			}`,
			Mutate: true,
			Parameters: []domain.PolicyParameters{
				{
					Name:     "exclude_namespace",
					Type:     "string",
					Required: false,
					Value:    "",
				},
				{
					Name:     "exclude_label_key",
					Type:     "string",
					Required: false,
					Value:    "",
				},
				{
					Name:     "exclude_label_value",
					Type:     "string",
					Required: false,
					Value:    "",
				},
			},
		},
		"runningAsRoot": {
			Name: "Container Running As Root",
			ID:   uuid.NewV4().String(),
			Code: `
			package magalix.advisor.podSecurity.runningAsRoot

			exclude_namespace := input.parameters.exclude_namespace
			exclude_label_key := input.parameters.exclude_label_key
			exclude_label_value := input.parameters.exclude_label_value

			# Check for missing securityContext.runAsNonRoot (missing in both, pod and container)
			violation[result] {
				not exclude_namespace == controller_input.metadata.namespace
				not exclude_label_value == controller_input.metadata.labels[exclude_label_key]

				controller_spec.securityContext
				not controller_spec.securityContext.runAsNonRoot
				not controller_spec.securityContext.runAsNonRoot == false

				some i
				containers := controller_spec.containers[i]
				containers.securityContext
				not containers.securityContext.runAsNonRoot
				not containers.securityContext.runAsNonRoot == false

				result = {
					"issue detected": true,
					"msg": sprintf("Container missing spec.template.spec.containers[%v].securityContext.runAsNonRoot while Pod spec.template.spec.securityContext.runAsNonRoot is not defined as well.", [i]),
					"violating_key": sprintf("spec.template.spec.containers[%v].securityContext", [i]),
				}
			}

			# Container security context
			# Check if containers.securityContext.runAsNonRoot exists and = false
			violation[result] {
				not exclude_namespace == controller_input.metadata.namespace
				not exclude_label_value == controller_input.metadata.labels[exclude_label_key]

				some i
				containers := controller_spec.containers[i]
				containers.securityContext
				containers.securityContext.runAsNonRoot == false

				result = {
					"issue detected": true,
					"msg": sprintf("Container spec.template.spec.containers[%v].securityContext.runAsNonRoot should be set to true", [i]),
					"violating_key": sprintf("spec.template.spec.containers[%v].securityContext.runAsNonRoot", [i]),
					"recommended_value": true,
				}
			}

			# Pod security context
			# Check if spec.securityContext.runAsNonRoot exists and = false
			violation[result] {
				not exclude_namespace == controller_input.metadata.namespace
				not exclude_label_value == controller_input.metadata.labels[exclude_label_key]

				controller_spec.securityContext
				controller_spec.securityContext.runAsNonRoot == false

				result = {
					"issue detected": true,
					"msg": "Pod spec.template.spec.securityContext.runAsNonRoot should be set to true",
					"violating_key": "spec.template.spec.securityContext.runAsNonRoot",
					"recommended_value": true,
				}
			}

			controller_input = input.review.object

			controller_spec = controller_input.spec.template.spec {
				contains(controller_input.kind, {"StatefulSet", "DaemonSet", "Deployment", "Job", "ReplicaSet"})
			} else = controller_input.spec {
				controller_input.kind == "Pod"
			} else = controller_input.spec.jobTemplate.spec.template.spec {
				controller_input.kind == "CronJob"
			}

			contains(kind, kinds) {
				kinds[_] = kind
			}`,
			Parameters: []domain.PolicyParameters{
				{
					Name:     "exclude_namespace",
					Type:     "string",
					Required: false,
					Value:    "",
				},
				{
					Name:     "exclude_label_key",
					Type:     "string",
					Required: false,
					Value:    "",
				},
				{
					Name:     "exclude_label_value",
					Type:     "string",
					Required: false,
					Value:    "",
				},
			},
		},
		"badPolicyCode": {
			Name: "Missing owner label in metadata",
			ID:   uuid.NewV4().String(),
			Code: `
			Not valid code^^
			}`,
			Parameters: []domain.PolicyParameters{
				{
					Name:     "exclude_namespace",
					Type:     "string",
					Required: false,
					Value:    "",
				},
				{
					Name:     "exclude_label_key",
					Type:     "string",
					Required: false,
					Value:    "",
				},
				{
					Name:     "exclude_label_value",
					Type:     "string",
					Required: false,
					Value:    "",
				},
			},
		},
		"replicaCount": {
			Name: "Minimum replica count",
			ID:   uuid.NewV4().String(),
			Code: `
			package weave.advisor.pods.replica_count
			import future.keywords.in

			min_replica_count := input.parameters.replica_count
			exclude_namespaces := input.parameters.exclude_namespaces
			exclude_label_key := input.parameters.exclude_label_key
			exclude_label_value := input.parameters.exclude_label_value

			controller_input := input.review.object

			violation[result] {
			isExcludedNamespace == false
			not exclude_label_value == controller_input.metadata.labels[exclude_label_key]
			not replicas >= min_replica_count
			result = {
				"issue detected": true,
				"msg": sprintf("Replica count must be greater than or equal to '%v'; found '%v'.", [min_replica_count, replicas]),
				"violating_key": violating_key,
				"recommended_value": min_replica_count,
			}
			}

			replicas := controller_input.spec.replicas {
				controller_input.kind in {"Deployment", "StatefulSet", "ReplicaSet", "ReplicationController"}
			} else := controller_input.spec.minReplicas {
				controller_input.kind == "HorizontalPodAutoscaler"
			}

			violating_key := "spec.replicas" {
				controller_input.kind in {"Deployment", "StatefulSet", "ReplicaSet", "ReplicationController"}
			} else := "spec.minReplicas" {
				controller_input.kind == "HorizontalPodAutoscaler"
			}

			isExcludedNamespace = true {
				controller_input.metadata.namespace
				controller_input.metadata.namespace in exclude_namespaces
			} else = false
			`,
			Parameters: []domain.PolicyParameters{
				{
					Name:     "replica_count",
					Type:     "integer",
					Required: true,
					Value:    4,
				},
				{
					Name:     "exclude_namespaces",
					Type:     "array",
					Required: true,
					Value:    nil,
				},
				{
					Name:     "exclude_label_key",
					Type:     "string",
					Required: false,
					Value:    "",
				},
				{
					Name:     "exclude_label_value",
					Type:     "string",
					Required: false,
					Value:    "",
				},
			},
		},
		"imageTagEnforced": {
			Name:    "Using latest image tag in container",
			Enforce: true,
			ID:      uuid.NewV4().String(),
			Code: `
		package magalix.advisor.images.image_tag_enforce

		image_tag := input.parameters.image_tag
		exclude_namespace := input.parameters.exclude_namespace
		exclude_label_key := input.parameters.exclude_label_key
		exclude_label_value := input.parameters.exclude_label_value

		violation[result] {
		not exclude_namespace == controller_input.metadata.namespace
		not exclude_label_value == controller_input.metadata.labels[exclude_label_key]
		some i
		containers = controller_spec.containers[i]
		splittedUrl = split(containers.image, "/")
		image = splittedUrl[count(splittedUrl)-1]
		not contains(image, ":")
		result = {
			"issue detected": true,
			"msg": "Image is not tagged",
			"violating_key": sprintf("spec.template.spec.containers[%v].image", [i])
		}
		}

		violation[result] {
		some i
		containers = controller_spec.containers[i]
		splittedUrl = split(containers.image, "/")
		image = splittedUrl[count(splittedUrl)-1]
		count(split(image, ":")) == 2
		[image_name, tag] = split(image, ":")
		tag == image_tag
		result = {
			"issue detected": true,
			"msg": sprintf("Image contains unapproved tag '%v'", [image_tag]),
			"image": image,
			"violating_key": sprintf("spec.template.spec.containers[%v].image", [i])
		}
		}

		violation[result] {
		some i
		containers = controller_spec.containers[i]
		splittedUrl = split(containers.image, "/")
		image = splittedUrl[count(splittedUrl)-1]
		count(split(image, ":")) == 3
		[image_name, port, tag] = split(image, ":")
		tag == image_tag
		result = {
			"issue detected": true,
			"msg": sprintf("Image contains unapproved tag:'%v'", [image_tag]),
			"image": image,
			"violating_key": sprintf("spec.template.spec.containers[%v].image", [i])
		}
		}

		# Controller input
		controller_input = input.review.object

		# controller_container acts as an iterator to get containers from the template
		controller_spec = controller_input.spec.template.spec {
		contains_kind(controller_input.kind, {"StatefulSet" , "DaemonSet", "Deployment", "Job"})
		} else = controller_input.spec {
		controller_input.kind == "Pod"
		} else = controller_input.spec.jobTemplate.spec.template.spec {
		controller_input.kind == "CronJob"
		}

		contains_kind(kind, kinds) {
		kinds[_] = kind
		}`,
			Mutate: true,
			Parameters: []domain.PolicyParameters{
				{
					Name:     "image_tag",
					Type:     "string",
					Required: true,
					Value:    "latest",
				},
				{
					Name:     "exclude_namespace",
					Type:     "string",
					Required: false,
					Value:    "",
				},
				{
					Name:     "exclude_label_key",
					Type:     "string",
					Required: false,
					Value:    "",
				},
				{
					Name:     "exclude_label_value",
					Type:     "string",
					Required: false,
					Value:    "",
				},
			},
		},
		"imageTagExcluded": {
			Name: "Using latest image tag in container",
			Exclude: domain.PolicyExclusions{
				Namespaces: []string{"unit-testing", "flux-system"},
				Resources:  []string{"unit-testing/nginx-deployment"},
				Labels: []map[string]string{
					{
						"app": "nginx",
					},
				},
			},
			Enforce: true,
			ID:      uuid.NewV4().String(),
			Code: `
		package magalix.advisor.images.image_tag_enforce

		image_tag := input.parameters.image_tag

		violation[result] {
		some i
		containers = controller_spec.containers[i]
		splittedUrl = split(containers.image, "/")
		image = splittedUrl[count(splittedUrl)-1]
		not contains(image, ":")
		result = {
			"issue detected": true,
			"msg": "Image is not tagged",
			"violating_key": sprintf("spec.template.spec.containers[%v].image", [i])
		}
		}

		violation[result] {
		some i
		containers = controller_spec.containers[i]
		splittedUrl = split(containers.image, "/")
		image = splittedUrl[count(splittedUrl)-1]
		count(split(image, ":")) == 2
		[image_name, tag] = split(image, ":")
		tag == image_tag
		result = {
			"issue detected": true,
			"msg": sprintf("Image contains unapproved tag '%v'", [image_tag]),
			"image": image,
			"violating_key": sprintf("spec.template.spec.containers[%v].image", [i])
		}
		}

		violation[result] {
		some i
		containers = controller_spec.containers[i]
		splittedUrl = split(containers.image, "/")
		image = splittedUrl[count(splittedUrl)-1]
		count(split(image, ":")) == 3
		[image_name, port, tag] = split(image, ":")
		tag == image_tag
		result = {
			"issue detected": true,
			"msg": sprintf("Image contains unapproved tag:'%v'", [image_tag]),
			"image": image,
			"violating_key": sprintf("spec.template.spec.containers[%v].image", [i])
		}
		}

		# Controller input
		controller_input = input.review.object

		# controller_container acts as an iterator to get containers from the template
		controller_spec = controller_input.spec.template.spec {
		contains_kind(controller_input.kind, {"StatefulSet" , "DaemonSet", "Deployment", "Job"})
		} else = controller_input.spec {
		controller_input.kind == "Pod"
		} else = controller_input.spec.jobTemplate.spec.template.spec {
		controller_input.kind == "CronJob"
		}

		contains_kind(kind, kinds) {
		kinds[_] = kind
		}`,
			Mutate: true,
			Parameters: []domain.PolicyParameters{
				{
					Name:     "image_tag",
					Type:     "string",
					Required: true,
					Value:    "latest",
				},
			},
		},
	}
)
