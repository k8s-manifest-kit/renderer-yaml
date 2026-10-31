package yaml_test

const podYAML = `
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    app: test-app
    component: frontend
spec:
  containers:
  - name: nginx
    image: nginx:latest
`

const configMapYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  labels:
    app: test-app
    component: backend
data:
  config.yaml: "port: 8080"
`

const multiDocYAML = `
apiVersion: v1
kind: Service
metadata:
  name: test-service
spec:
  ports:
  - port: 80
---
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
type: Opaque
data:
  password: cGFzc3dvcmQ=
`
