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