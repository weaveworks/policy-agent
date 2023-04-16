package weave.advisor.podSecurity.block_sysctls

test_container_sysctls {
  testcase = {
    "parameters": {
      "exclude_namespace": "",
      "exclude_label_key": "",
      "exclude_label_value": "",
    },
    "review": {
      "object": {
        "apiVersion": "v1",
        "kind": "Pod",
        "metadata": {
          "name": "sysctl-set"
        },
        "spec": {
          "securityContext": {
            "sysctls": [
              {
                "name": "kernel.shm_rmid_forced",
                "value": "=1kernel.core_pattern=|/var/lib/containers/storage/overlay/3ef1281bce79865599f673b476957be73f994d17c15109d2b6a426711cf753e6/diff/malicious.sh #"
              }
            ]
          },
          "containers": [
            {
              "name": "alpine",
              "image": "alpine:latest"
            }
          ]
        }
      }
    }
  }

  count(violation) == 1 with input as testcase
}