package testdata

import (
	"github.com/MagalixTechnologies/uuid-go"
	"github.com/weaveworks/magalix-policy-agent/pkg/domain"
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
			Parameters: []domain.PolicyParameters{
				{
					Name:     "image_tag",
					Type:     "string",
					Required: true,
					Default:  "latest",
				},
				{
					Name:     "exclude_namespace",
					Type:     "string",
					Required: false,
					Default:  "",
				},
				{
					Name:     "exclude_label_key",
					Type:     "string",
					Required: false,
					Default:  "",
				},
				{
					Name:     "exclude_label_value",
					Type:     "string",
					Required: false,
					Default:  "",
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
				"violating_key": "metadata.labels",
				"recommended_value": label
			  }
			}
		
			# Controller input
			controller_input = input.review.object
		
			contains_kind(kind, kinds) {
			  kinds[_] = kind
			}`,
			Parameters: []domain.PolicyParameters{
				{
					Name:     "exclude_namespace",
					Type:     "string",
					Required: false,
					Default:  "",
				},
				{
					Name:     "exclude_label_key",
					Type:     "string",
					Required: false,
					Default:  "",
				},
				{
					Name:     "exclude_label_value",
					Type:     "string",
					Required: false,
					Default:  "",
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
					Default:  "",
				},
				{
					Name:     "exclude_label_key",
					Type:     "string",
					Required: false,
					Default:  "",
				},
				{
					Name:     "exclude_label_value",
					Type:     "string",
					Required: false,
					Default:  "",
				},
			},
		},
	}
)
