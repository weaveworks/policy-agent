package weave.advisor.podSecurity.runningAsRoot

test_pod_runAsNonRoot_false {
  testcase = {
    "parameters": {
      "exclude_namespaces": [],
      "exclude_label_key": "",
      "exclude_label_value": "",
    },
    "review": {
      "object": {
        "apiVersion": "v1",
        "kind": "Pod",
        "metadata": {
          "name": "security-context-demo",
        },
        "spec": {
          "securityContext" : {
            "runAsNonRoot": false,
          },
          "containers": [
            {
              "securityContext" : {
                "runAsNonRoot": true,
                "runAsUser": 1000,
              },
              "name": "sec-ctx-demo",
              "image": "busybox",
              "command": [
                "sh",
                "-c",
                "sleep 1h"
              ],
            },
            {
              "securityContext" : {
                "runAsNonRoot": true,
                "runAsUser": 1000,
              },
              "name": "sec-ctx-demo2",
              "image": "busybox",
              "command": [
                "sh",
                "-c",
                "sleep 1h"
              ],
            }
          ]
        }
      }
    }
  }

  count(violation) == 1 with input as testcase
}