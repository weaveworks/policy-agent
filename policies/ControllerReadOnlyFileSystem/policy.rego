package weave.advisor.podSecurity.enforce_ro_fs

import future.keywords.in

read_only = input.parameters.read_only
exclude_namespaces := input.parameters.exclude_namespaces
exclude_label_key := input.parameters.exclude_label_key
exclude_label_value := input.parameters.exclude_label_value

violation[result] {
  isExcludedNamespace == false
  not exclude_label_value == controller_input.metadata.labels[exclude_label_key]
  some i
  containers := controller_spec.containers[i]
  root_fs := containers.securityContext.readOnlyRootFilesystem
  not root_fs == read_only
  result = {
    "issue detected": true,
    "msg": sprintf("readOnlyRootFilesystem should equal '%v'; detected '%v'", [read_only, root_fs]),
    "recommended_value": read_only,
    "violating_key": sprintf("spec.template.spec.containers[%v].securityContext.readOnlyRootFilesystem", [i]) 
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
}

isExcludedNamespace = true {
	controller_input.metadata.namespace
	controller_input.metadata.namespace in exclude_namespaces
} else = false
