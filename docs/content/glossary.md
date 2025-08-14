# Glossary

This glossary defines terms used in the NSO Operator documentation and the broader NSO and Kubernetes ecosystem.

## A

**Admin Credentials**  
Authentication information (username and password) required to access NSO administrative functions. Stored in Kubernetes Secrets for security.

**Anti-affinity**  
Kubernetes scheduling constraint that prevents pods from being scheduled on the same node, used for high availability deployments.

**API Reference**  
Documentation describing the structure and fields of Custom Resource Definitions (CRDs) used by the NSO Operator.

## B

**Blue-Green Deployment**  
Deployment strategy that maintains two identical environments (blue and green), switching traffic between them for zero-downtime updates.

## C

**Canary Deployment**  
Deployment strategy that gradually shifts traffic from an old version to a new version, allowing for testing with minimal risk.

**CDB (Configuration Database)**  
NSO's internal database that stores network device configurations and service models.

**ConfigMap**  
Kubernetes resource used to store non-confidential configuration data in key-value pairs.

**Controller**  
Kubernetes component that runs control loops, watching for changes to resources and taking actions to achieve the desired state.

**CRD (Custom Resource Definition)**  
Kubernetes API extension that allows you to define custom resources with custom schemas and behaviors.

**Custom Resource (CR)**  
An instance of a Custom Resource Definition, representing a desired state of a particular application or service.

## D

**Deployment**  
Kubernetes resource that manages a replicated set of pods, providing declarative updates and rollback capabilities.

**Device**  
Network equipment (routers, switches, firewalls, etc.) that NSO manages and configures.

**Docker Image**  
Packaged application that includes the NSO software and its dependencies, ready to run in containers.

## E

**Endpoint**  
Network address (IP and port) where a service can be reached.

**Environment Variable**  
Configuration values passed to containers at runtime, used to customize NSO behavior.

## F

**Finalizer**  
Kubernetes mechanism that prevents deletion of a resource until specific cleanup tasks are completed.

**FQDN (Fully Qualified Domain Name)**  
Complete domain name that specifies an exact location in the DNS hierarchy.

## G

**Git Repository**  
Version control system used to store and manage NSO packages and configurations.

**GitOps**  
Operational framework that uses Git as the single source of truth for declarative infrastructure and applications.

## H

**Health Check**  
Automated test to determine if an application or service is running correctly and can handle requests.

**Helm Chart**  
Package format for Kubernetes applications that simplifies deployment and management.

**High Availability (HA)**  
System design approach that ensures service remains available even when components fail.

## I

**Image Pull Secret**  
Kubernetes secret containing credentials for accessing private container registries.

**Ingress**  
Kubernetes resource that manages external access to services, typically HTTP/HTTPS routing.

**Init Container**  
Specialized container that runs before application containers in a pod, often used for setup tasks.

## J

**Job**  
Kubernetes resource that runs pods to completion, commonly used for batch processing and one-time tasks.

**JSON (JavaScript Object Notation)**  
Lightweight data interchange format used for configuration and API communication.

## K

**kubectl**  
Command-line tool for interacting with Kubernetes clusters.

**Kubernetes**  
Open-source container orchestration platform for automating deployment, scaling, and management of applications.

## L

**Label**  
Key-value pairs attached to Kubernetes objects, used for organization and selection.

**Label Selector**  
Query used to identify a set of objects based on their labels.

**Lifecycle**  
The complete process of creating, updating, and deleting an application or service.

**Liveness Probe**  
Health check that determines if a container is running properly; failed probes result in container restart.

**LoadBalancer**  
Kubernetes service type that provisions an external load balancer to distribute traffic.

## M

**Manifest**  
YAML or JSON file that describes the desired state of Kubernetes resources.

**Metrics**  
Quantitative measurements of system behavior, often used for monitoring and alerting.

**Multi-tenancy**  
Architecture that allows multiple users or teams to share the same infrastructure while maintaining isolation.

## N

**Namespace**  
Kubernetes mechanism for dividing cluster resources between multiple users or environments.

**NED (Network Element Driver)**  
NSO component that provides device-specific communication protocols and data models.

**NETCONF**  
Network Configuration Protocol, standardized protocol for network device configuration.

**Network Policy**  
Kubernetes resource that controls traffic flow between pods and external endpoints.

**NSO (Network Services Orchestrator)**  
Cisco's network automation and orchestration platform.

**NSO Operator**  
Kubernetes operator that manages NSO instances and package bundles.

## O

**Operator**  
Kubernetes extension that uses custom resources to manage applications and their components.

**Operator SDK**  
Framework for building Kubernetes operators using best practices and common patterns.

## P

**Package**  
NSO software component containing device drivers (NEDs), service models, or other functionality.

**Package Bundle**  
Collection of NSO packages managed as a single unit by the NSO Operator.

**Persistent Volume (PV)**  
Kubernetes resource representing storage that exists independently of pod lifecycle.

**Persistent Volume Claim (PVC)**  
Request for storage by a user, specifying size and access requirements.

**Pod**  
Smallest deployable unit in Kubernetes, containing one or more containers.

**Port Forward**  
kubectl command that creates a tunnel between local machine and Kubernetes cluster.

**Prometheus**  
Open-source monitoring system that collects and stores metrics as time series data.

## Q

**Quality of Service (QoS)**  
Classification system that determines resource allocation and scheduling priority for pods.

## R

**RBAC (Role-Based Access Control)**  
Kubernetes authorization mechanism that restricts access based on user roles.

**Readiness Probe**  
Health check that determines if a container is ready to receive traffic.

**Reconciliation**  
Process by which controllers compare desired state with actual state and take corrective actions.

**Replica**  
Copy of a pod running the same application, used for scaling and availability.

**ReplicaSet**  
Kubernetes resource that ensures a specified number of pod replicas are running.

**RESTCONF**  
REST-based protocol for accessing configuration and operational data defined by YANG models.

**Rolling Update**  
Deployment strategy that gradually replaces old instances with new ones, maintaining availability.

## S

**Secret**  
Kubernetes resource for storing sensitive information like passwords, tokens, and keys.

**Selector**  
Label query used to identify a set of Kubernetes objects.

**Service**  
Kubernetes resource that provides stable network endpoints for accessing pods.

**Service Account**  
Kubernetes identity used by pods to authenticate with the API server.

**ServiceMonitor**  
Prometheus Operator resource that defines how to scrape metrics from services.

**Spec**  
Specification section of a Kubernetes resource that describes the desired state.

**StatefulSet**  
Kubernetes resource for managing stateful applications with stable network identities.

**Status**  
Section of a Kubernetes resource that shows the current observed state.

## T

**TLS (Transport Layer Security)**  
Cryptographic protocol for secure communication over networks.

**Toleration**  
Kubernetes mechanism that allows pods to be scheduled on nodes with matching taints.

## U

**Update Strategy**  
Configuration that determines how updates are rolled out to running applications.

## V

**Volume**  
Directory accessible to containers in a pod, used for persistent storage or sharing data.

**Volume Mount**  
Configuration that makes a volume available at a specific path within a container.

## W

**Webhook**  
HTTP callback mechanism used for admission control and validation in Kubernetes.

**Workload**  
Application or service running in Kubernetes, typically managed by controllers like Deployments.

## Y

**YAML (YAML Ain't Markup Language)**  
Human-readable data serialization format commonly used for Kubernetes resource definitions.

**YANG**  
Data modeling language used for network configuration and management protocols.

## Z

**Zero-downtime Deployment**  
Deployment strategy that updates applications without service interruption.

---

## Related Documentation

For more detailed information about these concepts:

- [User Guide](user-guide/) - Practical usage of NSO Operator concepts
- [API Reference](api-reference/) - Detailed resource specifications
- [Kubernetes Glossary](https://kubernetes.io/docs/reference/glossary/) - Official Kubernetes terminology
- [NSO Documentation](https://developer.cisco.com/docs/nso/) - Comprehensive NSO concepts and features