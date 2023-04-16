package weave.advisor.podSecurity.runningAsRoot

import future.keywords.in

exclude_namespaces := input.parameters.exclude_namespaces
exclude_label_key := input.parameters.exclude_label_key
exclude_label_value := input.parameters.exclude_label_value

# Check for missing securityContext.runAsNonRoot (missing in both, pod and container)
violation[result] {
	isExcludedNamespace == false
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
	isExcludedNamespace == false
	not exclude_label_value == controller_input.metadata.labels[exclude_label_key]

	some i
	containers := controller_spec.containers[i]
	containers.securityContext
	containers.securityContext.runAsNonRoot == false

	result = {
		"issue detected": true,
		"msg": sprintf("Container spec.template.spec.containers[%v].securityContext.runAsNonRoot should be set to true ", [i]),
		"violating_key": sprintf("spec.template.spec.containers[%v].securityContext.runAsNonRoot", [i]),
		"recommended_value": true,
	}
}

# Pod security context
# Check if spec.securityContext.runAsNonRoot exists and = false
violation[result] {
	isExcludedNamespace == false
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
}

isExcludedNamespace = true {
	controller_input.metadata.namespace
	controller_input.metadata.namespace in exclude_namespaces
} else = false
