package weave.advisor.podSecurity.runningAsRoot

test_exclude_namespace {
  testcase = {
    "parameters": {
      "exclude_namespaces": ["allow-root"],
      "exclude_label_key": "",
      "exclude_label_value": "",
    },
    "review": {
      "object": {
        "apiVersion": "v1",
        "kind": "Pod",
        "metadata": {
          "name": "security-context-demo",
          "namespace": "allow-root",
        },
        "spec": {
          "securityContext" : {
            "runAsNonRoot": false
          },
          "containers": [
            {
              "securityContext" : {
                "runAsNonRoot": false,
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
                "runAsNonRoot": false,
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

  count(violation) == 0 with input as testcase
}