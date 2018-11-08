# Advanced Deployment

Dsiem support clustering mode for horizontal scalability. In this mode each instance of dsiem will run either as frontend or backend node, with NATS messaging in between to facilitate communication. The architecture is depicted in the following diagram.

![Advanced Architecture](/docs/images/advanced-arch.png)

## About the Architecture

* Frontend nodes is responsible for validating and parsing incoming normalized events from Logstash, and forwarding its results to backend nodes through NATS messaging system. Since frontends do not maintain states, Logstash can be easily configured to load balance between them (for example, using DNS round-robin or Kubernetes load balancer).

* Each backend node is assigned with a different set of (exclusive) directive rules to use for processing events. For example, a directive rule for detecting port scan maybe assigned only to backend A, and another directive that alert on SSH failed logins maybe assigned only to backend B. This simple way of distributing workload means that all the backends can work independently, using an in memory storage, and do not have to share states with each other.

* That however, does entail that *every* event sent from frontends will have to be delivered/broadcasted to all backends in order to evaluate them against all rules. This could potentially introduce network bottleneck from NATS to backends. It also means that the rules assigned to a failing backend will not be picked up by other nodes.

* For now we consider the above drawbacks acceptable and somewhat manageable (for instance through network configuration), given that the alternative method of distributing event processing between backends seem to require maintaining shared states: a pattern that will introduce much greater complexity and likely performance penalty.

## Configuration

* Example cluster mode configuration is provided <a href="https://github.com/defenxor/dsiem/blob/master/deployments/docker/docker-compose-cluster.yml">here</a>. To try it out just follow the [Installation](./Installation.md#using-docker-compose)  guide, and use the following command to execute `docker-compose up`:

    ```shell
    cd dsiem/deployments/docker && \
    docker-compose -f docker-compose-cluster.yml up
    ```

* Locations of all web interface endpoints (Kibana, Elasticsearch, Dsiem) are the same as in the standalone mode.

